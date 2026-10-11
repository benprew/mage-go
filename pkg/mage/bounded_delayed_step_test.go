package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestBoundedDelayedStepLifetime(t *testing.T) {
	for _, when := range []string{"before", "after", "skipped", "pending", "countered", "clone", "additional-combat"} {
		t.Run(when, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			if when == "after" || when == "additional-combat" {
				g.FireEvent(core.GameEvent{Type: core.EvtEndOfCombat, PlayerID: a.PlayerID()})
			}
			effect := AtNextThisTurn(EachEndOfCombat, GainLifeFor(You, Fixed(2)))
			if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if a.Life() != 20 || len(g.DelayedTriggers()) != 1 || g.DelayedTriggers()[0].Persistent {
				t.Fatal("schedule did not remain a one-shot")
			}
			clone := g.Clone()
			if when == "after" || when == "skipped" || when == "clone" {
				g.RunStepWithPriority(core.Cleanup)
				if len(g.DelayedTriggers()) != 0 {
					t.Fatal("untriggered schedule survived cleanup")
				}
				g.FireEvent(core.GameEvent{Type: core.EvtEndOfCombat, PlayerID: b.PlayerID()})
				g.ResolveStack()
				if a.Life() != 20 {
					t.Fatal("expired schedule triggered")
				}
			} else {
				g.FireEvent(core.GameEvent{Type: core.EvtEndOfCombat, PlayerID: b.PlayerID()})
				if a.Life() != 20 || len(g.DelayedTriggers()) != 0 {
					t.Fatal("body ran at detection or schedule remained")
				}
				if when == "pending" {
					g.triggers.ClearEndOfTurn()
				}
				g.PutTriggersOnStack()
				if when == "countered" {
					g.GetStack().RemoveByID(g.StackPeek().ID)
				}
				g.ResolveStack()
				want := 22
				if when == "countered" {
					want = 20
				}
				if a.Life() != want || b.Life() != 20 {
					t.Fatal("pending body or controller lost")
				}
				g.FireEvent(core.GameEvent{Type: core.EvtEndOfCombat, PlayerID: a.PlayerID()})
				g.ResolveStack()
				if a.Life() != want {
					t.Fatal("schedule repeated")
				}
			}
			if len(clone.DelayedTriggers()) != 1 {
				t.Fatal("clone schedule changed")
			}
			clone.RunStepWithPriority(core.EndCombat)
			if clone.GetPlayer(a.PlayerID()).Life() != 22 {
				t.Fatal("clone lost saved schedule")
			}
		})
	}
}

func TestBoundedDelayedStepDeclarations(t *testing.T) {
	for _, tc := range []struct {
		effect ResolvingEffect
		valid  bool
	}{
		{AtNextThisTurn(EachEndOfCombat, GainLifeFor(You, Fixed(1))), true},
		{AtNextThisTurn(nil, GainLifeFor(You, Fixed(1))), false},
		{AtNextThisTurn(EachEndOfCombat, nil), false},
		{AtNextThisTurn(EachEndOfCombat, Destroy(From(BindPermanent("missing")))), false},
	} {
		card := NewInstant("Bounded delay", "", Spell(tc.effect))
		if err := ValidateCardDefinition(card.Name(), card); (err == nil) != tc.valid {
			t.Fatalf("validation=%v, valid=%v", err, tc.valid)
		}
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	if err := ApplyEffect(g, AtNext(EachEndOfCombat, GainLifeFor(You, Fixed(1))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.RunStepWithPriority(core.Cleanup)
	if len(g.DelayedTriggers()) != 1 {
		t.Fatal("ordinary next-step behavior changed")
	}
	g.RunStepWithPriority(core.EndCombat)
	if a.Life() != 21 {
		t.Fatal("ordinary next-step schedule lost")
	}
}
