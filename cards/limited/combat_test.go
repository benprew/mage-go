package limited

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

// CR 509.1h — "An attacking creature is 'blocked' from the time a creature is
// declared as a blocker for it, and remains blocked even if all creatures
// blocking it are removed from combat."
// CR 510.1c — If no creatures are assigned to block it, it assigns no combat
// damage at all (unless it has trample).
func TestBlockedCreatureRemainsBlocked(t *testing.T) {
	t.Run("blocker killed before damage — attacker deals no damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")    // 3/3 attacker
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // 2/2 blocker
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")

		g.Attack(1, gametest.PlayerA, "Hill Giant")
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "Hill Giant")
		// Kill the blocker with Lightning Bolt after blockers declared, before damage.
		// Use FirstStrikeDamage step: executeOrderedActions fires before damage resolves,
		// and blocks were already declared in DeclareBlockers step.
		g.CastSpell(1, core.FirstStrikeDamage, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")

		g.StopAt(1, core.EndStep)
		g.Execute()

		// Blocker should be dead
		g.AssertGraveyardCount(gametest.PlayerA, "Lightning Bolt", 1)
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
		// Attacker was blocked — even though blocker is gone, it stays blocked
		// and deals no damage to the defending player (CR 509.1h + 510.1c)
		g.AssertLife(gametest.PlayerB, 20)
	})

	t.Run("blocker killed before damage — attacker with trample deals damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Craw Wurm")     // 6/4
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // 2/2 blocker
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Berserk")

		// Berserk grants trample and doubles power
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Berserk", "Craw Wurm")
		g.Attack(1, gametest.PlayerA, "Craw Wurm")
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "Craw Wurm")
		g.CastSpell(1, core.FirstStrikeDamage, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")

		g.StopAt(1, core.EndStep)
		g.Execute()

		// Blocker dead
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
		// Craw Wurm had 12 power (doubled by Berserk) with trample.
		// All blockers gone, so all 12 tramples through.
		g.AssertLife(gametest.PlayerB, 8)
	})
}
