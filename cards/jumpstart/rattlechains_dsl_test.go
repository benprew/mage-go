package jumpstart

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestRattlechainsEntryTarget(t *testing.T) {
	for _, scenario := range []string{"cast self", "enter self", "noncreature Spirit"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
			card, err := mage.CreateCard("Rattlechains")
			if err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			targetName, owner := card.Name(), gametest.PlayerA
			if scenario == "noncreature Spirit" {
				spirit := mage.NewEnchantment("Spirit permanent", "", mage.WithSubTypes("Spirit"))
				spirit.SetOwner(b.PlayerID())
				g.PutOnBattlefield(spirit, b.PlayerID())
				targetName, owner = spirit.Name(), gametest.PlayerB
			}
			g.ChooseTarget(gametest.PlayerA, targetName)
			if scenario == "cast self" {
				a.AddToHand(card)
				a.ManaPool().Add(core.Blue, 2)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatalf("cast without entry target: %v", err)
				}
				if len(g.StackPeek().Targets) != 0 {
					t.Fatal("creature spell has a target")
				}
				g.ResolveTopOfStack()
			} else {
				g.PutOnBattlefield(card, a.PlayerID())
				g.PutTriggersOnStack()
			}
			if g.StackSize() != 1 {
				t.Fatalf("entry trigger stack size = %d, want 1", g.StackSize())
			}
			g.AssertHasAbility(owner, targetName, core.Hexproof, false)
			g.ResolveStack()
			g.AssertHasAbility(owner, targetName, core.Hexproof, true)
			g.StopAt(2, core.Upkeep)
			g.Execute()
			g.AssertHasAbility(owner, targetName, core.Hexproof, false)
		})
	}
}
