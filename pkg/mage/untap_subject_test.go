package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestSubjectUntapRestriction(t *testing.T) {
	for _, change := range []string{"none", "zero", "move", "source-remove", "host-remove", "source-left", "host-return", "source-phase", "phase-cycle", "control", "clone"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			hostCard := NewCreature("Host", "", 2, 8)
			hostCard.SetOwner(b.PlayerID())
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			other := g.PutOnBattlefield(NewCreature("Other", "", 2, 8), b.PlayerID())
			c := NewEnchantment("Restriction", "", WithAbility(Static(PreventUntapDuringStep(AttachedPermanent, SourceHasCounters(core.Net, 1)))))
			c.AddSubType("Aura")
			c.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition(c.Name(), c); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(c, a.PlayerID())
			g.Attach(source.ID(), host.ID())
			g.MutablePermanent(source.ID()).Counters[core.Net] = 1
			clone := g.Clone()
			blocked, otherBlocked := true, false
			switch change {
			case "zero":
				g.MutablePermanent(source.ID()).Counters[core.Net] = 0
				blocked = false
			case "move":
				g.Attach(source.ID(), other.ID())
				blocked, otherBlocked = false, true
			case "source-remove":
				g.AddContinuousEffect(RemoveAllAbilities(source.ID(), core.Indefinite))
				blocked = false
			case "host-remove":
				g.AddContinuousEffect(RemoveAllAbilities(host.ID(), core.Indefinite))
			case "source-left":
				g.DestroyPermanent(g.FindPermanent(source.ID()))
				blocked = false
			case "host-return":
				g.BouncePermanentToHand(g.FindPermanent(host.ID()))
				b.RemoveFromHand(hostCard.ID())
				host = g.PutOnBattlefield(hostCard, b.PlayerID())
				blocked = false
			case "source-phase":
				g.PhaseOut(source.ID())
				blocked = false
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(host.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: host.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				g.SetActivePlayerIndex(0)
			case "clone":
				clone.MutablePermanent(source.ID()).Counters[core.Net] = 0
			}
			for _, p := range []*Permanent{g.FindPermanent(host.ID()), g.FindPermanent(other.ID())} {
				g.TapPermanent(p)
			}
			if change != "control" {
				g.SetActivePlayerIndex(1)
			}
			g.RunStepWithPriority(core.Untap)
			if g.FindPermanent(host.ID()).Tapped != blocked {
				t.Fatal("subject decision")
			}
			if change != "control" && g.FindPermanent(other.ID()).Tapped != otherBlocked {
				t.Fatal("attachment decision")
			}
			clone.TapPermanent(clone.FindPermanent(host.ID()))
			clone.SetActivePlayerIndex(1)
			clone.RunStepWithPriority(core.Untap)
			if clone.FindPermanent(host.ID()).Tapped != (change != "clone") {
				t.Fatal("clone decision")
			}
			if blocked && !g.UntapPermanent(g.FindPermanent(host.ID())) {
				t.Fatal("other effect prevented")
			}
		})
	}
}

func TestSubjectUntapDefinition(t *testing.T) {
	for _, tc := range []struct {
		name      string
		subject   PermanentRef
		condition StateCondition
		valid     bool
	}{
		{"source", This, SourceHasCounters(core.Net, 1), true},
		{"host", AttachedPermanent, PermanentIsTapped(This), true},
		{"nil subject", nil, SourceHasCounters(core.Net, 1), false},
		{"nil condition", This, nil, false},
		{"unbound", DeclarePermanentTarget("unbound", AnyPermanent), SourceHasCounters(core.Net, 1), false},
		{"negative", This, SourceHasCounters(core.Net, -1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewArtifact(tc.name, "", WithAbility(Static(PreventUntapDuringStep(tc.subject, tc.condition))))
			if err := ValidateCardDefinition(c.Name(), c); (err == nil) != tc.valid {
				t.Fatalf("validation %v", err)
			}
		})
	}
}

func TestSubjectUntapDecisionSnapshot(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewEnchantment("Restriction", "", WithAbility(Static(PreventUntapDuringStep(AttachedPermanent, PermanentIsTapped(This)))))
	c.AddSubType("Aura")
	source := g.PutOnBattlefield(c, a.PlayerID())
	host := g.PutOnBattlefield(NewCreature("Host", "", 2, 8), a.PlayerID())
	g.Attach(source.ID(), host.ID())
	g.TapPermanent(source)
	g.TapPermanent(g.FindPermanent(host.ID()))
	g.RunStepWithPriority(core.Untap)
	if g.FindPermanent(source.ID()).Tapped || !g.FindPermanent(host.ID()).Tapped {
		t.Fatal("condition changed during simultaneous untap")
	}
	g.RunStepWithPriority(core.Untap)
	if g.FindPermanent(host.ID()).Tapped {
		t.Fatal("condition did not refresh")
	}
}
