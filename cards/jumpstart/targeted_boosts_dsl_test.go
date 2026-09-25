package jumpstart

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestTargetedBoostsPartialLegality(t *testing.T) {
	for _, spellCase := range []struct {
		name  string
		power int
		life  int
	}{
		{"Dauntless Onslaught", 2, 0},
		{"Tandem Tactics", 1, 2},
	} {
		for _, allIllegal := range []bool{false, true} {
			scenario := "one returned target"
			if allIllegal {
				scenario = "all targets returned"
			}
			t.Run(spellCase.name+"/"+scenario, func(t *testing.T) {
				g := gametest.NewTestGame(t)
				a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
				first := mage.NewCreature("First target", "", 2, 2)
				second := mage.NewCreature("Second target", "", 2, 2)
				first.SetOwner(a.PlayerID())
				second.SetOwner(b.PlayerID())
				g.PutOnBattlefield(first, a.PlayerID())
				g.PutOnBattlefield(second, b.PlayerID())
				spell, err := mage.CreateCard(spellCase.name)
				if err != nil {
					t.Fatal(err)
				}
				spell.SetOwner(a.PlayerID())
				a.AddToHand(spell)
				a.ManaPool().Add(core.White, 3)
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{first.ID(), second.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
				g.PutOnBattlefield(first, a.PlayerID())
				if allIllegal {
					g.RemoveFromBattlefield(g.FindPermanent(second.ID()))
					g.PutOnBattlefield(second, b.PlayerID())
				}
				g.ResolveStack()
				g.AssertPowerToughness(gametest.PlayerA, first.Name(), 2, 2)
				power, toughness, life := 2+spellCase.power, 4, 20+spellCase.life
				if allIllegal {
					power, toughness, life = 2, 2, 20
				}
				g.AssertPowerToughness(gametest.PlayerB, second.Name(), power, toughness)
				g.AssertLife(gametest.PlayerA, life)
				g.AssertLife(gametest.PlayerB, 20)
				g.AssertGraveyardCount(gametest.PlayerA, spellCase.name, 1)
			})
		}
	}
}
