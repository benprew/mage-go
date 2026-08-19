package thedark

import (
	"testing"

	_ "github.com/benprew/mage-go/cards/limited"
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func bloodMoonManaProduction(permanent *mage.Permanent, color core.Color) int {
	total := 0
	for _, ability := range permanent.RuntimeAbilities {
		for _, production := range mage.ManaProductionsForAbility(ability) {
			if production.Color == color {
				total += production.Amount
			}
		}
	}
	return total
}

func TestBloodMoon(t *testing.T) {
	t.Run("nonbasic lands become Mountains", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Blood Moon")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Badlands")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()

		badlands := g.FindPermanentByName("Badlands", g.GetPlayer(gametest.PlayerA).PlayerID())
		if badlands == nil {
			t.Fatal("Badlands not found")
		}
		if !badlands.HasSubType("Mountain") || badlands.HasSubType("Swamp") {
			t.Fatalf("Badlands subtypes: Mountain=%v Swamp=%v", badlands.HasSubType("Mountain"), badlands.HasSubType("Swamp"))
		}
		if red, black := bloodMoonManaProduction(badlands, core.Red), bloodMoonManaProduction(badlands, core.Black); red != 1 || black != 0 {
			t.Fatalf("Badlands mana abilities: red=%d black=%d, want red=1 black=0", red, black)
		}

		forest := g.FindPermanentByName("Forest", g.GetPlayer(gametest.PlayerA).PlayerID())
		if forest == nil || !forest.HasSubType("Forest") || forest.HasSubType("Mountain") {
			t.Fatal("Blood Moon changed a basic Forest")
		}
	})

	t.Run("lands revert when Blood Moon leaves", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Blood Moon")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Badlands")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Disenchant")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Disenchant", "Blood Moon")
		g.StopAt(1, core.BeginCombat)
		g.Execute()

		badlands := g.FindPermanentByName("Badlands", g.GetPlayer(gametest.PlayerA).PlayerID())
		if badlands == nil {
			t.Fatal("Badlands not found")
		}
		if !badlands.HasSubType("Swamp") || !badlands.HasSubType("Mountain") {
			t.Fatal("Badlands did not regain its printed land subtypes")
		}
		if red, black := bloodMoonManaProduction(badlands, core.Red), bloodMoonManaProduction(badlands, core.Black); red != 1 || black != 1 {
			t.Fatalf("restored Badlands mana abilities: red=%d black=%d, want red=1 black=1", red, black)
		}
	})
}
