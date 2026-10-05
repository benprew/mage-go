package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRampageBlockedTransition(t *testing.T) {
	for _, entering := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		card := NewCreature("Rampage", "", 2, 4, WithAbility(RampageTrigger(2)), WithAbility(RampageTrigger(1)))
		card.SetOwner(a.PlayerID())
		p := g.PutOnBattlefield(card, a.PlayerID())
		g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
		first := g.PutOnBattlefield(NewCreature("First", "", 1, 1), b.PlayerID())
		if entering {
			g.PutOnBattlefieldBlocking(NewCreature("Entering", "", 1, 1), b.PlayerID(), p.ID())
		} else {
			g.GetCombat().AddBlocker(first.ID(), p.ID())
			g.fireNewlyBlockedAttackers(map[uuid.UUID]bool{})
		}
		g.PutTriggersOnStack()
		if g.StackSize() != 2 {
			t.Fatalf("entering=%v: want two separate triggers, got %d", entering, g.StackSize())
		}
		g.PutOnBattlefieldBlocking(NewCreature("Later", "", 1, 1), b.PlayerID(), p.ID())
		g.PutTriggersOnStack()
		if g.StackSize() != 2 {
			t.Fatal("already blocked source triggered again")
		}
		clone := g.Clone()
		g.ResolveStack()
		if g.FindPermanent(p.ID()).CurrentPower(g) != 5 {
			t.Fatal("bonus did not count blockers at resolution")
		}
		if clone.FindPermanent(p.ID()).CurrentPower(clone) != 2 {
			t.Fatal("clone changed")
		}
		clone.ResolveStack()
		if clone.FindPermanent(p.ID()).CurrentPower(clone) != 5 {
			t.Fatal("clone lost pending trigger")
		}
		g.RemoveFromCombat(first.ID())
		if g.FindPermanent(p.ID()).CurrentPower(g) != 5 {
			t.Fatal("resolved bonus changed with blockers")
		}
		g.RunStepWithPriority(core.Cleanup)
		if g.FindPermanent(p.ID()).CurrentPower(g) != 2 {
			t.Fatal("bonus did not expire")
		}
	}
}

func TestRampageOriginalIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Rampage", "", 2, 4, WithAbility(RampageTrigger(2)))
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
	for range 2 {
		g.PutOnBattlefieldBlocking(NewCreature("Blocker", "", 1, 1), b.PlayerID(), p.ID())
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("missing pending trigger")
	}
	g.BouncePermanentToHand(p)
	a.RemoveFromHand(card.ID())
	returned := g.PutOnBattlefield(card, a.PlayerID())
	g.GetCombat().AddAttacker(returned.ID(), b.PlayerID())
	for range 3 {
		blocker := g.PutOnBattlefield(NewCreature("New blocker", "", 1, 1), b.PlayerID())
		g.GetCombat().AddBlocker(blocker.ID(), returned.ID())
	}
	g.ResolveStack()
	if g.FindPermanent(returned.ID()).CurrentPower(g) != 2 {
		t.Fatal("old trigger boosted a new incarnation")
	}
}

func TestRampagePendingLifecycle(t *testing.T) {
	for _, scenario := range []string{"remove-blocker", "remove-source", "phase-source", "remove-ability", "control", "phase-after", "return-after", "copy"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Rampage", "", 2, 4, WithAbility(RampageTrigger(2)))
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
			first := g.PutOnBattlefieldBlocking(NewCreature("First", "", 1, 1), b.PlayerID(), p.ID())
			g.PutOnBattlefieldBlocking(NewCreature("Second", "", 1, 1), b.PlayerID(), p.ID())
			g.PutTriggersOnStack()
			switch scenario {
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "remove-blocker":
				g.RemoveFromCombat(first.ID())
			case "remove-source":
				g.RemoveFromCombat(p.ID())
			case "phase-source":
				g.PhaseOut(p.ID())
			case "remove-ability":
				g.AddContinuousEffect(RemoveAllAbilities(p.ID(), core.EndOfTurn))
			case "control":
				if err := ApplyEffect(g, Until(core.EndOfTurn, ChangeController(This, You)), p.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			g.ResolveStack()
			if scenario == "phase-source" {
				g.PhaseIn([]uuid.UUID{p.ID()})
			}
			want := 4
			if scenario == "copy" {
				want = 6
			}
			if scenario == "remove-blocker" || scenario == "remove-source" || scenario == "phase-source" || scenario == "control" {
				want = 2
			}
			if g.FindPermanent(p.ID()).CurrentPower(g) != want {
				t.Fatalf("power=%d want %d", g.FindPermanent(p.ID()).CurrentPower(g), want)
			}
			if scenario == "phase-after" {
				g.PhaseOut(p.ID())
				g.PhaseIn([]uuid.UUID{p.ID()})
				if g.FindPermanent(p.ID()).CurrentPower(g) != 4 {
					t.Fatal("phasing lost bonus")
				}
			}
			if scenario == "return-after" {
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				a.RemoveFromHand(card.ID())
				returned := g.PutOnBattlefield(card, a.PlayerID())
				if returned.CurrentPower(g) != 2 {
					t.Fatal("bonus followed return")
				}
			}
		})
	}
}
