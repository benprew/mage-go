package arabian

import (
	"testing"

	"github.com/mage/mage"
	_ "github.com/mage/mage/cards" // register base cards for test creatures
)

func TestAbuJafar(t *testing.T) {
	t.Run("attacking_destroys_blockers_when_dies", func(t *testing.T) {
		// Abu Ja'far (0/1) attacks, blocked by a 2/2. Abu Ja'far dies to combat
		// damage, trigger destroys the blocker.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Abu Ja'far")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Grizzly Bears") // 2/2
		g.Attack(1, mage.PlayerA, "Abu Ja'far")
		g.Block(1, mage.PlayerB, "Grizzly Bears", "Abu Ja'far")
		g.StopAt(1, mage.PostcombatMain)
		g.Execute()
		g.AssertGraveyardCount(mage.PlayerA, "Abu Ja'far", 1)
		g.AssertGraveyardCount(mage.PlayerB, "Grizzly Bears", 1)
	})

	t.Run("blocking_destroys_attacker_when_dies", func(t *testing.T) {
		// Opponent attacks with a 3/3, Abu Ja'far (0/1) blocks and dies,
		// trigger destroys the attacker.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Abu Ja'far")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Centaur Courser") // 3/3
		g.Attack(2, mage.PlayerB, "Centaur Courser")
		g.Block(2, mage.PlayerA, "Abu Ja'far", "Centaur Courser")
		g.StopAt(2, mage.PostcombatMain)
		g.Execute()
		g.AssertGraveyardCount(mage.PlayerA, "Abu Ja'far", 1)
		g.AssertGraveyardCount(mage.PlayerB, "Centaur Courser", 1)
	})

	t.Run("no_combat_no_effect", func(t *testing.T) {
		// Abu Ja'far is destroyed outside combat — nothing else dies.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Abu Ja'far")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Grizzly Bears")
		g.AddCard(mage.ZoneHand, mage.PlayerB, "Lightning Bolt")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerB, "Lightning Bolt", "Abu Ja'far")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(mage.PlayerA, "Abu Ja'far", 1)
		g.AssertPermanentCount(mage.PlayerB, "Grizzly Bears", 1)
	})
}
