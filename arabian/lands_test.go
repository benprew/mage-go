package arabian

import (
	"testing"

	"github.com/mage/mage"
	_ "github.com/mage/mage/cards" // register base cards for test creatures
)

func TestDesert(t *testing.T) {
	t.Run("desert_does_damage", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Savannah Lions")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Desert")
		g.Attack(2, mage.PlayerB, "Savannah Lions")
		g.ActivateAbility(2, mage.EndCombat, mage.PlayerA, "Desert", "Savannah Lions")
		g.StopAt(2, mage.EndStep)
		g.Execute()
		// Savannah Lions should die!
		g.AssertPermanentCount(mage.PlayerB, "Savannah Lions", 0)
		g.AssertGraveyardCount(mage.PlayerB, "Savannah Lions", 1)

	})
}
