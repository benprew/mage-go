package jumpstart

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"

	_ "git.sr.ht/~cdcarter/mage-go/cards/limited"
)

func TestToweringTitan_EntersWithCountersFromOthersToughness(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Towering Titan")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 6)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")    // 3/3
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Towering Titan")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Towering Titan", core.P1P1, 5)
	g.AssertPowerToughness(gametest.PlayerA, "Towering Titan", 5, 5)
}

func TestToweringTitan_DiesWhenNoOtherCreatures(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Towering Titan")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 6)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Towering Titan")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Towering Titan", 0)
	g.AssertGraveyardCount(gametest.PlayerA, "Towering Titan", 1)
}

func TestToweringTitan_OpponentCreaturesDoNotCount(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Towering Titan")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 6)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")    // 3/3 (opponent)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Towering Titan")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Towering Titan", core.P1P1, 2)
}

func TestToweringTitan_BranchingEvolutionDoublesCounters(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Towering Titan")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest", 6)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Branching Evolution")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Towering Titan")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertCounterCount(gametest.PlayerA, "Towering Titan", core.P1P1, 4)
}
