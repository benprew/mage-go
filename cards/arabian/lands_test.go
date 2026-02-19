package arabian

import (
	"testing"

	"github.com/mage/mage/pkg/mage/core"
	"github.com/mage/mage/pkg/mage/gametest"
	_ "github.com/mage/mage/cards/limited" // register base cards for test creatures
)

func TestDesert(t *testing.T) {
	t.Run("desert_does_damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Savannah Lions")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Desert")
		g.Attack(2, gametest.PlayerB, "Savannah Lions")
		g.ActivateAbility(2, core.EndCombat, gametest.PlayerA, "Desert", "Savannah Lions")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// Savannah Lions should die!
		g.AssertPermanentCount(gametest.PlayerB, "Savannah Lions", 0)
		g.AssertGraveyardCount(gametest.PlayerB, "Savannah Lions", 1)

	})
}
