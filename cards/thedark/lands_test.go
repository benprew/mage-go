package thedark

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestMazeOfIth(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		name := "unblocked"
		if blocked {
			name = "blocked"
		}
		t.Run(name, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Maze of Ith")
			g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
			g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
			g.Attack(1, gametest.PlayerA, "Grizzly Bears")
			if blocked {
				g.Block(1, gametest.PlayerB, "Grizzly Bears", "Grizzly Bears")
			}
			g.ActivateAbility(1, core.DeclareBlockers, gametest.PlayerA, "Maze of Ith", "Grizzly Bears")
			g.StopAt(1, core.PostcombatMain)
			g.Execute()
			g.AssertLife(gametest.PlayerB, 20)
			g.AssertTapped(gametest.PlayerA, "Maze of Ith", true)
			g.AssertTapped(gametest.PlayerA, "Grizzly Bears", false)
			g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 2)
			g.AssertPowerToughness(gametest.PlayerB, "Grizzly Bears", 2, 2)
		})
	}
}
