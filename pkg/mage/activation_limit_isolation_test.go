package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestActivationLimitIsolation(t *testing.T) {
	for _, max := range []int{1, 2} {
		for _, change := range []string{"clone", "return", "control", "phase"} {
			t.Run(string(rune('0'+max))+"/"+change, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				card := NewArtifact("Limited source", "", WithAbility(NewAction(ActionActivated, WithEffect(GainLifeFor(You, Fixed(1))), WithMaxActivationsPerTurn(max))))
				card.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(card, a.PlayerID())
				fork := g.Clone()
				for range max {
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
				}
				switch change {
				case "clone":
					for range max {
						if err := fork.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
							t.Fatal("clone shares limit", err)
						}
					}
				case "return":
					g.BouncePermanentToHand(g.FindPermanent(p.ID()))
					a.RemoveFromHand(card.ID())
					p = g.PutOnBattlefield(card, a.PlayerID())
					for range max {
						if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
							t.Fatal("new incarnation shares limit", err)
						}
					}
				case "control":
					g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
					if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, nil); err == nil {
						t.Fatal("control change reset limit")
					}
				case "phase":
					g.PhaseIn(g.PhaseOut(p.ID()))
				}
				payer := a
				if change == "control" {
					payer = b
				}
				if err := g.ActivateAbilityByIndex(payer.PlayerID(), p.ID(), 0, nil); err == nil {
					t.Fatal("limit exceeded")
				}
				if card.Abilities()[0].(*ActionDefinition).activationsThisTurn != 0 {
					t.Fatal("runtime changed card declaration")
				}
				g.ResolveStack()
				if fork.GetPlayer(a.PlayerID()).Life() != 20 {
					t.Fatal("resolution changed clone")
				}
			})
		}
	}
}
