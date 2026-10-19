package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAttachmentTargetQuery(t *testing.T) {
	for _, state := range []string{"legal", "host control", "aura control", "host shroud", "aura shroud", "move", "wrong type", "stale", "missing host", "aura return", "host return", "phase", "cycle", "source leaves", "copy", "clone"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			host := g.PutOnBattlefield(NewCreature("Host", "", 2, 4), a.PlayerID())
			other := g.PutOnBattlefield(NewCreature("Other", "", 2, 4), b.PlayerID())
			aura := g.PutOnBattlefield(NewAura("Aura", "", WithCastTarget(TargetPermanent())), b.PlayerID())
			g.Attach(aura.ID(), host.ID())
			binding := DeclarePermanentTarget("aura", And(HasSubType("Aura"), IsAttachedTo(Or(IsCreature, IsLand))), TargetAttachedToYouControl())
			source := g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(Activate(Costs(), Destroy(binding), WithTargets(binding)))), a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{aura.ID()}); err != nil {
				t.Fatal(err)
			}
			plan, ok := InspectPermanentTarget(binding)
			if !ok {
				t.Fatal("attachment target opaque")
			}
			obj := g.StackPeek()
			want := true
			switch state {
			case "host control":
				g.AddControlEffect(ControlEffectSpec{TargetID: host.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				want = false
			case "aura control":
				g.AddControlEffect(ControlEffectSpec{TargetID: aura.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "host shroud":
				host.GrantBaseAttr(core.Shroud)
			case "aura shroud":
				aura.GrantBaseAttr(core.Shroud)
				want = false
			case "move":
				g.Attach(aura.ID(), other.ID())
				want = false
			case "wrong type":
				host.RevokeBaseAttr(core.AttrIsCreature)
				host.GrantBaseAttr(core.AttrIsArtifact)
				want = false
			case "stale":
				aura.attachedIncarnation = uuid.New()
				want = false
			case "missing host":
				aura.AttachedTo = uuid.New()
				want = false
			case "aura return":
				g.RemoveFromBattlefield(aura)
				aura = g.PutOnBattlefield(aura.Card, b.PlayerID())
				g.Attach(aura.ID(), host.ID())
				want = false
			case "host return":
				g.RemoveFromBattlefield(host)
				g.PutOnBattlefield(host.Card, a.PlayerID())
				want = false
			case "phase":
				g.PhaseOut(host.ID())
				want = false
			case "cycle":
				g.PhaseIn(g.PhaseOut(host.ID()))
			case "source leaves":
				g.RemoveFromBattlefield(source)
			case "copy":
				obj = g.CopyStackObjectDirect(obj, a.PlayerID(), false)
			case "clone":
				clone := g.Clone()
				clone.Attach(aura.ID(), other.ID())
				if got, known := plan.Select(clone, clone.StackPeek()); !known || len(got) != 0 {
					t.Fatal("clone attachment ignored")
				}
			}
			native := slices.Contains(binding.Possible(obj.Controller, g.resolutionTargetSource(obj), g), aura.ID())
			if state == "aura return" {
				native = false
			}
			if native != want {
				t.Fatalf("native %v want %v", native, want)
			}
			for _, game := range []*Game{g, g.Clone()} {
				got, known := plan.Select(game, obj)
				if !known || (len(got) == 1) != want {
					t.Fatalf("selection %v/%v want %v", got, known, want)
				}
			}
			binding.spec.(*PermanentTarget).Filters[0].fn = func(*Permanent, *Game) bool { panic("query called filter") }
			if got, known := plan.Select(g, obj); !known || (len(got) == 1) != want {
				t.Fatal("callback-free query differs")
			}
		})
	}
}

func TestAttachmentTargetQueryOpaque(t *testing.T) {
	trap := NewPermanentFilter("opaque", func(*Permanent, *Game) bool { panic("opaque host called") })
	for _, filter := range []PermanentFilter{IsAttachedTo(trap), IsAttachedTo(Or(IsCreature, trap)), And(IsAttachedTo(IsCreature), trap)} {
		if _, ok := InspectPermanentTarget(DeclarePermanentTarget("aura", filter)); ok {
			t.Fatal("partial host declaration accepted")
		}
	}
}

func TestAttachmentTargetQueryTrees(t *testing.T) {
	g, a, b := newSpellCopyGame()
	creature := g.PutOnBattlefield(NewCreature("Host", "", 2, 4), a.PlayerID())
	land := g.PutOnBattlefield(NewLand("Land"), b.PlayerID())
	artifact := g.PutOnBattlefield(NewArtifact("Artifact", ""), a.PlayerID())
	aura := g.PutOnBattlefield(NewAura("Aura", "", WithCastTarget(TargetPermanent())), b.PlayerID())
	for _, host := range []*Permanent{creature, land, artifact} {
		g.Attach(aura.ID(), host.ID())
		for _, tc := range []struct {
			filter PermanentFilter
			want   bool
		}{
			{IsAttachedTo(Or(IsCreature, IsLand)), host != artifact},
			{And(HasSubType("Aura"), IsAttachedTo(And(IsCreature, HasPowerEqual(2)))), host == creature},
			{Or(IsLand, IsAttachedTo(Not(IsCreature))), host != creature},
		} {
			binding := DeclarePermanentTarget("aura", tc.filter)
			card := NewInstant("Source", "")
			obj := &StackObject{ID: uuid.New(), Card: card, SourceID: card.ID(), Controller: a.PlayerID(), Targets: []uuid.UUID{aura.ID()}, TargetSpecs: []Target{binding.targetSpec()}}
			g.PushStack(obj)
			plan, ok := InspectPermanentTarget(binding)
			if !ok {
				t.Fatal("host tree opaque")
			}
			native := slices.Contains(binding.Possible(a.PlayerID(), card, g), aura.ID())
			if native != tc.want {
				t.Fatal("native host tree mismatch")
			}
			binding.spec.(*PermanentTarget).Filters[0].fn = func(*Permanent, *Game) bool { panic("host tree callback") }
			got, known := plan.Select(g, obj)
			if !known || (len(got) == 1) != tc.want {
				t.Fatal("host tree selection differs")
			}
		}
	}
}

func TestAttachmentTargetQueryCollectionBoundary(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	host := g.PutOnBattlefield(NewCreature("Host", "", 2, 4), a.PlayerID())
	aura := g.PutOnBattlefield(NewAura("Aura", ""), a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	filter := IsAttachedTo(IsCreature)
	if _, ok := InspectPermanentCollection(MatchingPermanents(filter)); ok {
		t.Fatal("context-free attachment query accepted")
	}
	plan, ok := InspectContextualPermanentCollection(MatchingPermanents(filter))
	if !ok {
		t.Fatal("contextual attachment query lost")
	}
	filter.fn = func(*Permanent, *Game) bool { panic("collection called filter") }
	got := plan.Select(&EffectContext{Game: g, Controller: a.PlayerID()})
	if len(got) != 1 || got[0].ID != aura.ID() {
		t.Fatal("wrong attachment collection", got)
	}
	g.MutablePermanent(aura.ID()).attachedIncarnation = uuid.New()
	if got := plan.Select(&EffectContext{Game: g, Controller: a.PlayerID()}); len(got) != 0 {
		t.Fatal("collection accepted stale host")
	}
}
