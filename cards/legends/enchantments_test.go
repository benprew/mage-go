package legends

import (
	"testing"

	"github.com/mage/mage/pkg/mage/core"
	"github.com/mage/mage/pkg/mage/gametest"
)

func TestSeeker(t *testing.T) {
	t.Run("enchanted creature cannot be blocked by non-artifact non-white creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Seeker")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3 red
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Seeker", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Hill Giant", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Hill Giant is red and not artifact — can't block enchanted Bears
		g.AssertLife(gametest.PlayerB, 18)
	})

	t.Run("enchanted creature can be blocked by white creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Seeker")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Keepers of the Faith") // 2/3 white
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Seeker", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Keepers of the Faith", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Keepers of the Faith is white — can block
		g.AssertLife(gametest.PlayerB, 20)
	})
}

func TestMarblePriest(t *testing.T) {
	t.Run("Wall combat damage to Marble Priest is prevented", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		// Use a Marble Priest (3/3) attacking into a Wall of Opposition (0/6)
		// Pump the wall's power to 4 via its {1}: +1/+0 ability so it would be lethal
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Marble Priest") // 3/3
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wall of Opposition") // 0/6 Wall, {1}: +1/+0
		// Pump 4 times during begin combat on PlayerA's turn (PlayerB activates)
		g.ActivateAbility(1, core.BeginCombat, gametest.PlayerB, "Wall of Opposition")
		g.ActivateAbility(1, core.BeginCombat, gametest.PlayerB, "Wall of Opposition")
		g.ActivateAbility(1, core.BeginCombat, gametest.PlayerB, "Wall of Opposition")
		g.ActivateAbility(1, core.BeginCombat, gametest.PlayerB, "Wall of Opposition")
		g.Attack(1, gametest.PlayerA, "Marble Priest")
		g.Block(1, gametest.PlayerB, "Wall of Opposition", "Marble Priest")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Wall of Opposition is 4/6 — would deal 4 damage (lethal to 3/3) without prevention
		// With prevention, Marble Priest survives
		g.AssertPermanentCount(gametest.PlayerA, "Marble Priest", 1)
	})
}
