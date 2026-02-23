package arabian

import (
	"testing"

	"github.com/mage/mage/pkg/mage/core"
	"github.com/mage/mage/pkg/mage/gametest"
	_ "github.com/mage/mage/cards/limited" // register base cards for test creatures
)

func TestOasis(t *testing.T) {
	t.Run("prevents_1_damage_from_desert", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Savannah Lions") // 2/1
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Oasis")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Desert")
		g.Attack(1, gametest.PlayerA, "Savannah Lions")
		// Apply Oasis prevention first, then Desert damage
		g.ActivateAbility(1, core.EndCombat, gametest.PlayerA, "Oasis", "Savannah Lions")
		g.ActivateAbility(1, core.EndCombat, gametest.PlayerB, "Desert", "Savannah Lions")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// Savannah Lions should survive — 1 damage prevented
		g.AssertPermanentCount(gametest.PlayerA, "Savannah Lions", 1)
	})
}

func TestElephantGraveyard(t *testing.T) {
	t.Run("regenerates_elephant", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "War Elephant") // 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Elephant Graveyard")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		// Set up regeneration shield before the bolt
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Elephant Graveyard", "War Elephant")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "War Elephant")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// War Elephant should survive via regeneration
		g.AssertPermanentCount(gametest.PlayerA, "War Elephant", 1)
	})

	t.Run("cannot_target_non_elephant", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Elephant Graveyard")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Elephant Graveyard", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Activation should fail silently — not a valid target
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}

func TestIslandOfWakWak(t *testing.T) {
	t.Run("sets_flyer_power_to_0", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island of Wak-Wak")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Serra Angel") // 4/4 flying
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerB, "Island of Wak-Wak", "Serra Angel")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Power set to 0, toughness unchanged (4)
		g.AssertPowerToughness(gametest.PlayerA, "Serra Angel", 0, 4)
	})

	t.Run("cannot_target_non_flyer", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island of Wak-Wak")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // no flying
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerB, "Island of Wak-Wak", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Can't target non-flyer — should remain 2/2
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 2)
	})
}

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
