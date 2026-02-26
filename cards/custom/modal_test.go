package custom

import (
	"testing"

	_ "github.com/mage/mage/cards/limited" // register base cards (Plains)
	"github.com/mage/mage/pkg/mage/core"
	"github.com/mage/mage/pkg/mage/gametest"
)

func TestModalActivatedAbility_Mode1(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Modal Test Artifact")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
	g.ChooseMode(gametest.PlayerA, 0) // Mode 1: deal 1 damage
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Modal Test Artifact")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 19)
}

func TestModalActivatedAbility_Mode2(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Modal Test Artifact")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
	g.ChooseMode(gametest.PlayerA, 1) // Mode 2: gain 1 life
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Modal Test Artifact")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 21)
}
