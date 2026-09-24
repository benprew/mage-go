package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestAttachedPermanentResolutionLifetime(t *testing.T) {
	for _, change := range []string{"move", "move then leave", "host returns before source", "leave", "return", "return twice", "host returns", "turn cleanup", "phase source", "phase host", "clone"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			aura := NewAura("Attachment", "", WithAbility(Activate(Costs(), Sequence(AddCountersTo(AttachedPermanent, Charge, Fixed(1)), GainLifeFor(You, Fixed(2))))))
			first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
			for _, card := range []Card{first, second, aura} {
				card.SetOwner(a.PlayerID())
				g.PutOnBattlefield(card, a.PlayerID())
			}
			g.Attach(aura.ID(), first.ID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), aura.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			wantFirst, wantSecond := 1, 0
			switch change {
			case "move", "move then leave":
				g.Attach(aura.ID(), second.ID())
				wantFirst, wantSecond = 0, 1
				if change == "move then leave" {
					g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
				}
			case "leave", "return", "return twice", "host returns", "turn cleanup", "clone":
				g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
				if change == "return" || change == "return twice" {
					g.PutOnBattlefield(aura, a.PlayerID())
					g.Attach(aura.ID(), second.ID())
					if change == "return twice" {
						g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
					}
				}
				if change == "host returns" {
					g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
					g.PutOnBattlefield(first, a.PlayerID())
					wantFirst = 0
				}
				if change == "turn cleanup" {
					g.zones.ClearEndOfTurn()
				}
				if change == "clone" {
					original := g
					g = g.Clone()
					g.PutOnBattlefield(aura, a.PlayerID())
					g.Attach(aura.ID(), second.ID())
					g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
					original.ResolveStack()
					if original.FindPermanent(first.ID()).Counters[Charge] != 1 {
						t.Fatal("clone changed original attachment history")
					}
				}
			case "host returns before source":
				g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
				g.PutOnBattlefield(first, a.PlayerID())
				g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
				wantFirst = 0
			case "phase source":
				g.PhaseOut(aura.ID())
			case "phase host":
				g.PhaseOut(first.ID())
				wantFirst = 0
			}
			if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			g.ResolveStack()
			if g.FindPermanentIncludingPhased(first.ID()).Counters[Charge] != uint8(2*wantFirst) || g.FindPermanent(second.ID()).Counters[Charge] != uint8(2*wantSecond) {
				t.Fatal("attachment did not retain the required incarnation")
			}
			if g.GetPlayer(a.PlayerID()).Life() != 22 || g.GetPlayer(b.PlayerID()).Life() != 22 {
				t.Fatal("copy lost its controller or absent attachment stopped the sequence")
			}
		})
	}
}
