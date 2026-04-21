package arabian

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
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
		g.SetCoinFlipResults([]bool{true})                           // win
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
		g.SetCoinFlipResults([]bool{false}) // lose
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

func TestEbonyHorse(t *testing.T) {
	t.Run("untaps_and_removes_from_combat", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ebony Horse")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant") // 3/3
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Hill Giant")
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "Hill Giant")
		// Activate Ebony Horse on Hill Giant after blocks declared
		g.ActivateAbility(1, core.DeclareBlockers, gametest.PlayerA, "Ebony Horse", "Hill Giant")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// Hill Giant removed from combat — no damage dealt, both survive
		g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 1)
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
		g.AssertLife(gametest.PlayerB, 20)
	})
}

func TestCityInABottle(t *testing.T) {
	t.Run("sacrifices other Arabian Nights permanents when it enters", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Flying Carpet") // Arabian Nights artifact
		g.AddCard(core.ZoneHand, gametest.PlayerA, "City in a Bottle")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "City in a Bottle")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Flying Carpet is Arabian Nights — should be sacrificed
		g.AssertPermanentCount(gametest.PlayerB, "Flying Carpet", 0)
		// City in a Bottle itself stays
		g.AssertPermanentCount(gametest.PlayerA, "City in a Bottle", 1)
	})

	t.Run("non-Arabian permanents are unaffected", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // not Arabian
		g.AddCard(core.ZoneHand, gametest.PlayerA, "City in a Bottle")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "City in a Bottle")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Grizzly Bears stays
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
	})

	t.Run("prevents_casting_Arabian_Nights_spells", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "City in a Bottle")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Flying Carpet") // Arabian Nights artifact
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		// Try to cast an Arabian Nights spell — should be blocked
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Flying Carpet")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		// Flying Carpet should still be in hand (cast blocked)
		g.AssertPermanentCount(gametest.PlayerB, "Flying Carpet", 0)
	})

	t.Run("non_Arabian_spells_still_castable", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "City in a Bottle")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears") // not Arabian
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Grizzly Bears")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		// Grizzly Bears should be on battlefield (non-Arabian, not blocked)
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestSandalsOfAbdallah(t *testing.T) {
	t.Run("grants_islandwalk", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sandals of Abdallah")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Sandals of Abdallah", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Hill Giant", "Grizzly Bears")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// Bears have islandwalk, defender controls Island — unblockable
		g.AssertLife(gametest.PlayerB, 18)
	})

	t.Run("destroys_self_when_creature_dies", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sandals of Abdallah")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Sandals of Abdallah", "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Bears die → Sandals should be destroyed too
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
		g.AssertPermanentCount(gametest.PlayerA, "Sandals of Abdallah", 0)
	})
}

func TestJeweledBird(t *testing.T) {
	t.Run("antes_bird_returns_other_ante_cards_draws", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Jeweled Bird")
		// Player A has a card in ante (simulating the initial ante)
		g.AddCard(core.ZoneAnte, gametest.PlayerA, "Grizzly Bears")
		// Use a non-land card so autoPlayLands doesn't steal it from hand
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Jeweled Bird")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Jeweled Bird should be in ante
		g.AssertAnteCount(gametest.PlayerA, "Jeweled Bird", 1)
		// Original ante card should be in graveyard
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
		// Player drew Hill Giant from Bird's effect
		g.AssertHandCount(gametest.PlayerA, "Hill Giant", 1)
		// Bird is no longer on battlefield
		g.AssertPermanentCount(gametest.PlayerA, "Jeweled Bird", 0)
	})

	t.Run("works_with_empty_ante", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Jeweled Bird")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Jeweled Bird")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Bird in ante, drew a card
		g.AssertAnteCount(gametest.PlayerA, "Jeweled Bird", 1)
		g.AssertHandCount(gametest.PlayerA, "Hill Giant", 1)
		g.AssertPermanentCount(gametest.PlayerA, "Jeweled Bird", 0)
	})
}

func TestPyramids(t *testing.T) {
	t.Run("mode_1_destroys_aura_on_land", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Pyramids")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Psychic Venom")
		// Enchant opponent's Forest with Psychic Venom
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Psychic Venom", "Forest")
		// Use Pyramids to destroy the aura
		g.ChooseMode(gametest.PlayerA, 0) // Mode 1: destroy aura on land
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Pyramids", "Psychic Venom")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Psychic Venom destroyed, Forest still there
		g.AssertPermanentCount(gametest.PlayerA, "Psychic Venom", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Forest", 1)
	})

	t.Run("mode_2_prevents_land_destruction", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Pyramids")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Desert Twister")
		// Use Pyramids to protect the Forest
		g.ChooseMode(gametest.PlayerA, 1) // Mode 2: prevent land destruction
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Pyramids", "Forest")
		// Try to destroy the Forest
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Desert Twister", "Forest")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Forest survives thanks to Pyramids
		g.AssertPermanentCount(gametest.PlayerA, "Forest", 1)
	})
}

func TestAladdinsLamp(t *testing.T) {
	t.Run("replaces_draw_with_filtered_draw", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Aladdin's Lamp")
		for i := 0; i < 3; i++ {
			g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		}
		// Library: top to bottom — put a different card on top so replacement
		// produces a different result than a normal draw.
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Lightning Bolt")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant")
		// Activate lamp with X=3 during upkeep of turn 3 (PlayerA's second turn)
		// so the draw replacement fires on turn 3's draw step
		g.ActivateAbilityWithX(3, core.Upkeep, gametest.PlayerA, "Aladdin's Lamp", 3)
		// TestPlayer picks first candidate from the top 3 — Lightning Bolt
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		// Draw replacement: look at top 3, TestPlayer chooses first (Lightning Bolt),
		// rest go to bottom in random order
		g.AssertHandCount(gametest.PlayerA, "Lightning Bolt", 1)
	})

	t.Run("replacement_expires_at_end_of_turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Aladdin's Lamp")
		for i := 0; i < 3; i++ {
			g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		}
		// Library: several cards
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Lightning Bolt")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Shivan Dragon")
		// Activate lamp in turn 1 main phase — but don't draw this turn
		// (draw step already passed). The replacement should expire at end of turn.
		g.ActivateAbilityWithX(1, core.PrecombatMain, gametest.PlayerA, "Aladdin's Lamp", 3)
		// Stop at turn 3 draw step after cleanup clears the replacement
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		// Turn 3 draw should be a normal draw (Lightning Bolt, top of library)
		// because the replacement expired at end of turn 1
		g.AssertHandCount(gametest.PlayerA, "Lightning Bolt", 1)
	})
}

func TestJandorsRing(t *testing.T) {
	t.Run("discard_last_drawn_to_draw", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Jandor's Ring")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		// Put cards in library: first will be drawn in draw step on turn 3
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		// Turn 3 (PlayerA's second turn): draws Hill Giant, then activate Ring:
		// discard Hill Giant (last drawn), draw Grizzly Bears
		g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Jandor's Ring")
		g.StopAt(3, core.BeginCombat)
		g.Execute()
		// Hill Giant should be in graveyard (discarded as cost)
		g.AssertGraveyardCount(gametest.PlayerA, "Hill Giant", 1)
		// Grizzly Bears should be in hand (drawn by Ring)
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}
