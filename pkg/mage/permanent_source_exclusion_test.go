package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentSourceExclusion(t *testing.T) {
	for _, state := range []string{"live", "departed", "returned", "phase", "phase cycle", "control", "copy", "clone"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			binding := DeclarePermanentTarget("other", IsCreature, TargetOtherThanSource())
			card := NewCreature("Source", "", 2, 3, WithAbility(Activate(Costs(), Destroy(binding), WithTargets(binding))))
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			victim := g.PutOnBattlefield(NewCreature("Victim", "", 4, 5), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			obj := g.StackPeek()
			plan, ok := InspectPermanentTarget(binding)
			if !ok {
				t.Fatal("source exclusion is opaque")
			}
			switch state {
			case "departed", "returned":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if state == "returned" {
					source = g.PutOnBattlefield(card, b.PlayerID())
				}
			case "phase":
				g.PhaseOut(source.ID())
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(source.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				obj = g.CopyStackObjectDirect(obj, b.PlayerID(), false)
			case "clone":
				clone := g.Clone()
				clone.PhaseOut(victim.ID())
				if got, known := plan.Select(clone, clone.StackPeek()); !known || len(got) != 0 {
					t.Fatal("clone ignored target phasing")
				}
			}
			if got, known := plan.Select(g, obj); !known || len(got) != 1 || got[0].ID != victim.ID() {
				t.Fatalf("other target %v/%v", got, known)
			}
			if state == "departed" || state == "phase" {
				return
			}
			obj.Targets[0] = source.ID()
			obj.targetIncarnations = map[uuid.UUID]uuid.UUID{source.ID(): source.incarnationID}
			obj.TargetZones = map[uuid.UUID]core.Zone{source.ID(): core.ZoneBattlefield}
			want := state == "returned"
			sourceView := g.resolutionTargetSource(obj)
			possible := binding.Possible(obj.Controller, sourceView, g)
			if slices.Contains(possible, source.ID()) != want {
				t.Fatal("native exclusion merged source visits")
			}
			if got, known := plan.Select(g, obj); !known || (len(got) == 1) != want {
				t.Fatalf("self target %v/%v want %v", got, known, want)
			}
			g.ResolveStack()
			if (g.FindPermanent(source.ID()) == nil) != want {
				t.Fatal("resolution differs from query")
			}
		})
	}
}

func TestPermanentSourceExclusionNativeVisits(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	card := NewCreature("Source", "", 2, 3)
	source := g.PutOnBattlefield(card, a.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: source.ID(), TargetSource: card, IsAbility: true, Controller: a.PlayerID()}
	g.PushStack(obj)
	binding := DeclarePermanentTarget("other", IsCreature, TargetOtherThanSource())
	if slices.Contains(binding.Possible(a.PlayerID(), g.resolutionTargetSource(obj), g), source.ID()) {
		t.Fatal("source targeted itself")
	}
	g.RemoveFromBattlefield(source)
	returned := g.PutOnBattlefield(card, a.PlayerID())
	if !slices.Contains(binding.Possible(a.PlayerID(), g.resolutionTargetSource(obj), g), returned.ID()) {
		t.Fatal("returned source inherited old exclusion")
	}
}

func TestPermanentSourceExclusionUnknown(t *testing.T) {
	trap := NewPermanentFilter("opaque", func(*Permanent, *Game) bool { panic("query ran opaque filter") })
	for _, filter := range []PermanentFilter{trap, Or(AnyPermanent, trap), And(IsCreature, trap)} {
		if _, ok := InspectPermanentTarget(DeclarePermanentTarget("other", filter, TargetOtherThanSource())); ok {
			t.Fatal("partial exclusion accepted")
		}
	}
	g, p, obj, _ := targetQueryFixture(t)
	binding := DeclarePermanentTarget("other", IsLand, TargetOtherThanSource())
	obj.TargetSpecs[0] = binding.targetSpec()
	plan, ok := InspectPermanentTarget(binding)
	if !ok {
		t.Fatal("exclusion rejected")
	}
	obj.IsAbility = true
	obj.sourceIncarnation = uuid.Nil
	if _, known := plan.Select(g, obj); known {
		t.Fatal("missing source visit guessed")
	}
	obj.IsAbility = false
	binding.spec.(*PermanentTarget).Filters[0].fn = func(*Permanent, *Game) bool { panic("query ran filter") }
	if got, known := plan.Select(g, obj); !known || len(got) != 1 || got[0].ID != p.ID() {
		t.Fatal("retained filter lost")
	}
}
