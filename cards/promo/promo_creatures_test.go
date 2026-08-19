package promo

import (
	"testing"

	_ "github.com/benprew/mage-go/cards/limited"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestWindseekerCentaurVigilance(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Windseeker Centaur")
	g.Attack(1, gametest.PlayerA, "Windseeker Centaur")
	g.StopAt(1, core.PostcombatMain)
	g.Execute()
	g.AssertTapped(gametest.PlayerA, "Windseeker Centaur", false)
}

func TestGiantBadgerGetsPlusTwoPlusTwoWhenItBlocks(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Giant Badger")
	g.Attack(1, gametest.PlayerA, "Hill Giant")
	g.Block(1, gametest.PlayerB, "Giant Badger", "Hill Giant")
	g.StopAt(2, core.PrecombatMain)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Hill Giant", 1)
	g.AssertPermanentCount(gametest.PlayerB, "Giant Badger", 1)
	g.AssertPowerToughness(gametest.PlayerB, "Giant Badger", 2, 2)
}
