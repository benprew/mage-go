package promo

import (
	"testing"

	_ "github.com/benprew/mage-go/cards/limited"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestArenaOpponentChoosesTheirCreature(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Arena")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mons's Goblin Raiders")
	g.ChooseTarget(gametest.PlayerB, "Hill Giant")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Arena", "Grizzly Bears", "Mons's Goblin Raiders")
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	g.AssertTapped(gametest.PlayerA, "Arena", true)
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertTapped(gametest.PlayerB, "Hill Giant", true)
	g.AssertTapped(gametest.PlayerB, "Grizzly Bears", false)
}

func TestArenaCreaturesFightSimultaneously(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Arena")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mons's Goblin Raiders")
	g.ChooseTarget(gametest.PlayerB, "Grizzly Bears")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Arena", "Grizzly Bears", "Mons's Goblin Raiders")
	g.StopAt(1, core.BeginCombat)
	g.Execute()

	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
}
