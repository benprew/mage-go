package arabian

import (
	"testing"

	"github.com/mage/mage/pkg/mage/core"
	"github.com/mage/mage/pkg/mage/gametest"
	_ "github.com/mage/mage/cards/limited" // register base cards for test creatures
)

func TestBazaarOfBaghdad(t *testing.T) {
	t.Run("draw_2_discard_3", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Bazaar of Baghdad")
		// Start with 3 non-land cards in hand (lands get auto-played)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		// Put 2 cards in library to draw
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Bazaar of Baghdad")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Started with 3 Bears, drew 2 Giants (=5), discarded 3 (=2 remain)
		// AI discards first 3 chosen → 2 Hill Giants remain
		g.AssertHandCount(gametest.PlayerA, "Hill Giant", 2)
	})
}

func TestLibraryOfAlexandria(t *testing.T) {
	t.Run("draws_card_with_7_in_hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Library of Alexandria")
		// Put exactly 7 non-land cards in hand
		for i := 0; i < 7; i++ {
			g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		}
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Library of Alexandria")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Drew 1 Hill Giant
		g.AssertHandCount(gametest.PlayerA, "Hill Giant", 1)
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 7)
	})

	t.Run("does_nothing_without_7_in_hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Library of Alexandria")
		// Only 3 non-land cards in hand
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Library of Alexandria")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Should not draw — only 3 in hand, not 7
		g.AssertHandCount(gametest.PlayerA, "Hill Giant", 0)
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 3)
	})
}

func TestCityOfBrass(t *testing.T) {
	t.Run("tapping_for_mana_deals_1_damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "City of Brass")
		// Explicitly tap City of Brass for mana
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "City of Brass")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Tapping City of Brass for mana should deal 1 damage to controller
		g.AssertLife(gametest.PlayerA, 19)
	})
}

func TestDiamondValley(t *testing.T) {
	t.Run("gain_life_equal_to_toughness", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Diamond Valley")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant") // 3/3
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Diamond Valley")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Sacrificed Hill Giant (3 toughness), gain 3 life
		g.AssertLife(gametest.PlayerA, 23)
		g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 0)
	})
}

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
