package arabian

import (
	"testing"

	"github.com/mage/mage/pkg/mage/core"
	"github.com/mage/mage/pkg/mage/gametest"
)

func TestAladdinsRing(t *testing.T) {
	t.Run("deals_4_to_creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Aladdin's Ring")
		// Need 8 mana to activate
		for i := 0; i < 8; i++ {
			g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		}
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // 2/2
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Aladdin's Ring", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	})

	t.Run("deals_4_to_player", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Aladdin's Ring")
		for i := 0; i < 8; i++ {
			g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		}
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Aladdin's Ring", "PlayerB")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 16)
	})
}

func TestJandorsSaddlebags(t *testing.T) {
	t.Run("untaps_tapped_creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Jandor's Saddlebags")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		for i := 0; i < 3; i++ {
			g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		}
		// Attack with bears to tap them
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		// Untap them with saddlebags in postcombat
		g.ActivateAbility(1, core.PostcombatMain, gametest.PlayerA, "Jandor's Saddlebags", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Grizzly Bears", false)
	})
}

func TestBottleOfSuleiman(t *testing.T) {
	t.Run("win_flip_creates_djinn_token", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Bottle of Suleiman")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains") // {1} to activate
		g.CoinFlipResults = []bool{true}                            // win
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Bottle of Suleiman")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Bottle sacrificed
		g.AssertPermanentCount(gametest.PlayerA, "Bottle of Suleiman", 0)
		// 5/5 Djinn token created
		g.AssertPermanentCount(gametest.PlayerA, "Djinn", 1)
		// No damage taken
		g.AssertLife(gametest.PlayerA, 20)
	})

	t.Run("lose_flip_deals_5_damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Bottle of Suleiman")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.CoinFlipResults = []bool{false} // lose
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Bottle of Suleiman")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Bottle sacrificed
		g.AssertPermanentCount(gametest.PlayerA, "Bottle of Suleiman", 0)
		// No token
		g.AssertPermanentCount(gametest.PlayerA, "Djinn", 0)
		// Took 5 damage
		g.AssertLife(gametest.PlayerA, 15)
	})
}

func TestFlyingCarpet(t *testing.T) {
	t.Run("grants_flying_until_eot", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Flying Carpet")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wall of Stone") // 0/8 defender, no flying
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Flying Carpet", "Grizzly Bears")
		// Bears now have flying, attack — Wall can't block
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Wall of Stone", "Grizzly Bears")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// Wall can't block flyer — damage goes through
		g.AssertLife(gametest.PlayerB, 18)
	})
}
