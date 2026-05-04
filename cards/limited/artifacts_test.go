package limited

import (
	"testing"

	_ "git.sr.ht/~cdcarter/mage-go/cards/arabian"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

// Tests for cards registered in alpha_artifacts.go (and a few
// from alpha_spells.go / alpha_enchantments.go that are closely related).

// ---------------------------------------------------------------------------
// Artifacts
// ---------------------------------------------------------------------------

func TestWinterOrb(t *testing.T) {
	t.Run("limits_land_untap_to_one", func(t *testing.T) {
		// Winter Orb: Players can't untap more than one land during their untap steps.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Winter Orb")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		// Tap all lands by activating their mana abilities.
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Plains")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Island")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Mountain")
		// Turn 3 is PlayerA's next untap step. Only 1 land should untap.
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		// With Winter Orb, at most 1 land untaps; 2 should remain tapped.
		tapped := 0
		playerAID := g.AllPlayers()[0].PlayerID()
		for _, perm := range g.AllBattlefield() {
			if perm.Controller == playerAID && perm.HasType(core.TypeLand) && perm.Tapped {
				tapped++
			}
		}
		if tapped < 2 {
			t.Errorf("Winter Orb should keep 2 lands tapped, only %d tapped", tapped)
		}
	})

	t.Run("affects_opponent_too", func(t *testing.T) {
		// Winter Orb affects all players, not just the controller.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Winter Orb")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Swamp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerB, "Swamp")
		g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerB, "Mountain")
		g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerB, "Forest")
		g.StopAt(4, core.PrecombatMain)
		g.Execute()
		tapped := 0
		playerBID := g.AllPlayers()[1].PlayerID()
		for _, perm := range g.AllBattlefield() {
			if perm.Controller == playerBID && perm.HasType(core.TypeLand) && perm.Tapped {
				tapped++
			}
		}
		if tapped < 2 {
			t.Errorf("Winter Orb should limit opponent too; only %d lands tapped, want >= 2", tapped)
		}
	})

	t.Run("creatures_untap_normally", func(t *testing.T) {
		// Winter Orb only restricts lands; creatures untap normally.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Winter Orb")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Serra Angel")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
		g.Attack(1, gametest.PlayerA, "Hill Giant")
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Hill Giant", false)
	})

	t.Run("no_restriction_when_tapped", func(t *testing.T) {
		// When Winter Orb is tapped, its restriction doesn't apply.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Winter Orb")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Icy Manipulator")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		// Tap Winter Orb itself with Icy Manipulator.
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Icy Manipulator", "Winter Orb")
		// Tap all lands.
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Plains")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Island")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Mountain")
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		// Winter Orb is tapped, so all lands should untap normally.
		g.AssertTapped(gametest.PlayerA, "Plains", false)
		g.AssertTapped(gametest.PlayerA, "Island", false)
		g.AssertTapped(gametest.PlayerA, "Mountain", false)
	})
}

func TestMeekstone(t *testing.T) {
	t.Run("prevents_big_creatures_from_untapping", func(t *testing.T) {
		// Meekstone: Creatures with power 3 or greater don't untap during their
		// controller's untap step.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Meekstone")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant") // 3/3
		// Attack with Hill Giant to tap it.
		g.Attack(1, gametest.PlayerA, "Hill Giant")
		// Turn 3 is PlayerA's next untap step.
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		// Hill Giant (power 3) should NOT untap with Meekstone.
		g.AssertTapped(gametest.PlayerA, "Hill Giant", true)
	})

	t.Run("small_creatures_untap_normally", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Meekstone")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		// Grizzly Bears (power 2) should untap normally.
		g.AssertTapped(gametest.PlayerA, "Grizzly Bears", false)
	})
}

func TestHowlingMine(t *testing.T) {
	t.Run("both_players_draw_extra_card", func(t *testing.T) {
		// Howling Mine: At the beginning of each player's draw step, that player
		// draws an additional card.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Howling Mine")
		for range 10 {
			g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
			g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Grizzly Bears")
		}
		// Turn 1 doesn't draw for first player; check turn 3 (PlayerA's second turn).
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		// By turn 3, PlayerA has drawn: turn 3 draw = 1 (normal) + 1 (Mine) = 2.
		// PlayerB drew on turn 2: 1 (normal) + 1 (Mine) = 2.
		playerA := g.AllPlayers()[0]
		playerB := g.AllPlayers()[1]
		if len(playerA.Hand()) < 2 {
			t.Errorf("Howling Mine should give PlayerA extra draw; hand has %d cards, want >= 2", len(playerA.Hand()))
		}
		if len(playerB.Hand()) < 2 {
			t.Errorf("Howling Mine should give PlayerB extra draw; hand has %d cards, want >= 2", len(playerB.Hand()))
		}
	})

	t.Run("no_extra_draw_when_tapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Howling Mine")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Icy Manipulator")
		for range 10 {
			g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
			g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Forest")
		}
		// Tap Howling Mine before PlayerA's draw step.
		g.ActivateAbility(1, core.Upkeep, gametest.PlayerB, "Icy Manipulator", "Howling Mine")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Howling Mine is tapped, so no extra draw. Normal draw of 1.
		playerA := g.AllPlayers()[0]
		if len(playerA.Hand()) > 1 {
			t.Errorf("Tapped Howling Mine should not give extra draw; hand has %d cards, want 1", len(playerA.Hand()))
		}
	})
}

func TestForcefield(t *testing.T) {
	t.Run("reduces_unblocked_damage_to_one", func(t *testing.T) {
		// Forcefield: {1}: If an unblocked creature would deal combat damage to you,
		// prevent all but 1 of that damage.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forcefield")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Craw Wurm") // 6/4
		g.ActivateAbility(1, core.DeclareBlockers, gametest.PlayerB, "Forcefield")
		g.Attack(1, gametest.PlayerA, "Craw Wurm")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Forcefield should reduce 6 unblocked damage to 1.
		g.AssertLife(gametest.PlayerB, 19)
	})
}

func TestForcefieldBlockedCreature(t *testing.T) {
	t.Run("does_not_reduce_blocked_damage", func(t *testing.T) {
		// Forcefield only prevents unblocked combat damage.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forcefield")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Craw Wurm")     // 6/4
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // 2/2
		g.ActivateAbility(1, core.DeclareBlockers, gametest.PlayerB, "Forcefield")
		g.Attack(1, gametest.PlayerA, "Craw Wurm")
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "Craw Wurm")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Craw Wurm is blocked, so Forcefield doesn't apply. No player damage.
		g.AssertLife(gametest.PlayerB, 20)
	})

	t.Run("does_not_affect_1_power_creature", func(t *testing.T) {
		// A 1/1 unblocked creature deals 1 damage; Forcefield allows 1 through, so no change.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forcefield")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Llanowar Elves") // 1/1
		g.ActivateAbility(1, core.DeclareBlockers, gametest.PlayerB, "Forcefield")
		g.Attack(1, gametest.PlayerA, "Llanowar Elves")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 19)
	})

	t.Run("no_effect_without_activation", func(t *testing.T) {
		// If not activated, full damage goes through.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forcefield")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Craw Wurm") // 6/4
		g.Attack(1, gametest.PlayerA, "Craw Wurm")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 14)
	})
}

func TestTheHive(t *testing.T) {
	t.Run("creates_wasp_token", func(t *testing.T) {
		// The Hive: {5}, {T}: Create a 1/1 colorless Insect artifact creature
		// token with flying named Wasp.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "The Hive")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "The Hive")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Should have created a 1/1 Wasp token.
		g.AssertPermanentCount(gametest.PlayerA, "Wasp", 1)
	})

	t.Run("token_has_flying", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "The Hive")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "The Hive")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Wasp", core.Flying, true)
	})

	t.Run("token_is_artifact_creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "The Hive")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "The Hive")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		perm := g.FindPermanentByName("Wasp", g.AllPlayers()[0].PlayerID())
		if perm == nil {
			t.Fatal("Wasp token not found")
		}
		if !perm.HasType(core.TypeArtifact) {
			t.Error("Wasp should be an artifact")
		}
		if !perm.HasType(core.TypeCreature) {
			t.Error("Wasp should be a creature")
		}
		if perm.CurrentPower(g.Game) != 1 || perm.CurrentToughness(g.Game) != 1 {
			t.Errorf("Wasp should be 1/1, got %d/%d", perm.CurrentPower(g.Game), perm.CurrentToughness(g.Game))
		}
	})

	t.Run("token_can_attack", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "The Hive")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "The Hive")
		// Token has summoning sickness on turn 1; attack on turn 3.
		g.Attack(3, gametest.PlayerA, "Wasp")
		g.StopAt(3, core.EndCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 19)
	})
}

func TestCopyArtifact(t *testing.T) {
	t.Run("enters_as_copy_of_artifact", func(t *testing.T) {
		// Copy Artifact: You may have Copy Artifact enter the battlefield as a copy
		// of any artifact on the battlefield, except it's an enchantment in addition
		// to its other types.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sol Ring")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Copy Artifact")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Copy Artifact", "Sol Ring")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Should have 2 Sol Rings (original + copy).
		solCount := 0
		playerAID := g.AllPlayers()[0].PlayerID()
		for _, perm := range g.AllBattlefield() {
			if perm.Controller == playerAID && perm.Name() == "Sol Ring" {
				solCount++
			}
		}
		if solCount < 2 {
			t.Errorf("Copy Artifact should create copy of Sol Ring; found %d, want 2", solCount)
		}
	})

	t.Run("is_also_enchantment", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sol Ring")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Copy Artifact")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Copy Artifact", "Sol Ring")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// The copy becomes "Sol Ring" but is also an Enchantment.
		// Find the Sol Ring that has the Enchantment type.
		playerAID := g.AllPlayers()[0].PlayerID()
		found := false
		for _, perm := range g.AllBattlefield() {
			if perm.Controller == playerAID && perm.Name() == "Sol Ring" && perm.HasType(core.TypeEnchantment) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Copy Artifact should be an Enchantment in addition to Artifact")
		}
	})

	t.Run("copy_is_still_artifact", func(t *testing.T) {
		// The copy should retain the Artifact type from the original.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sol Ring")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Copy Artifact")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Copy Artifact", "Sol Ring")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		playerAID := g.AllPlayers()[0].PlayerID()
		found := false
		for _, perm := range g.AllBattlefield() {
			if perm.Controller == playerAID && perm.Name() == "Sol Ring" &&
				perm.HasType(core.TypeEnchantment) && perm.HasType(core.TypeArtifact) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Copy Artifact should be both Artifact and Enchantment")
		}
	})

	t.Run("copy_has_mana_ability", func(t *testing.T) {
		// The copy should have the same activated abilities as the original.
		// Activate the copy's mana ability to verify it produces mana.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sol Ring")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Copy Artifact")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Copy Artifact", "Sol Ring")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Find the copy (Sol Ring with Enchantment type) and verify it has abilities
		playerAID := g.AllPlayers()[0].PlayerID()
		for _, perm := range g.AllBattlefield() {
			if perm.Controller == playerAID && perm.Name() == "Sol Ring" && perm.HasType(core.TypeEnchantment) {
				if len(perm.RuntimeAbilities) == 0 {
					t.Errorf("Copy Artifact should have copied Sol Ring's abilities")
				}
				return
			}
		}
		t.Errorf("Copy not found")
	})
}

func TestLivingLands(t *testing.T) {
	t.Run("forests_become_creatures", func(t *testing.T) {
		// Living Lands: All Forests are 1/1 creatures. They're still lands.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Living Lands")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Forest should be a 1/1 creature.
		perm := g.FindPermanentByName("Forest", g.AllPlayers()[0].PlayerID())
		if perm == nil {
			t.Fatal("Forest not found")
		}
		if !perm.HasType(core.TypeCreature) {
			t.Errorf("Living Lands should make Forest a creature")
		}
		g.AssertPowerToughness(gametest.PlayerA, "Forest", 1, 1)
	})

	t.Run("forests_can_attack", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Living Lands")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.Attack(1, gametest.PlayerA, "Forest")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Animated Forest should deal 1 damage.
		g.AssertLife(gametest.PlayerB, 19)
	})

	t.Run("forests_still_tap_for_mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Living Lands")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Forest")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		perm := g.FindPermanentByName("Forest", g.AllPlayers()[0].PlayerID())
		if perm == nil {
			t.Fatal("Forest not found")
		}
		if !perm.HasType(core.TypeLand) {
			t.Errorf("Living Lands Forest should still be a land")
		}
	})
}

func TestManaFlare(t *testing.T) {
	t.Run("doubles_mana_from_lands", func(t *testing.T) {
		// Mana Flare: Whenever a player taps a land for mana, that player adds
		// one additional mana of any type that land produced.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mana Flare")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Mountain")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Mountain tapped for {R}. Mana Flare adds another {R}.
		// Auto-mana adds 5R. Mountain = 1R + 1R (Mana Flare) = 2R. Total = 7R.
		pool := g.AllPlayers()[0].ManaPool()
		if pool.CountProducedThisTurn(core.Red) < 7 {
			t.Errorf("Mana Flare should double land mana; expected >= 7 red, got %d", pool.CountProducedThisTurn(core.Red))
		}
	})

	t.Run("any_color_land_bonus_matches_chosen_color", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mana Flare")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "City of Brass")
		// Script the player to choose Blue when City of Brass asks for a color
		g.ChooseManaColor(gametest.PlayerA, core.Blue)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "City of Brass")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// City tapped for {U} (chosen). Mana Flare matches produced -> adds {U}.
		// Auto-mana adds 5R from Mountains. City = 1U + 1U (Mana Flare) = 2U.
		pool := g.AllPlayers()[0].ManaPool()
		if pool.CountProducedThisTurn(core.Blue) < 2 {
			t.Errorf("Mana Flare should add one mana of chosen type; expected >= 2 blue, got %d", pool.CountProducedThisTurn(core.Blue))
		}
	})

	t.Run("any_color_land_adds_only_one_bonus", func(t *testing.T) {
		// Even though City of Brass can produce any color, Mana Flare adds
		// exactly one mana of the type produced — not one of each.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mana Flare")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "City of Brass")
		g.ChooseManaColor(gametest.PlayerA, core.Green)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "City of Brass")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Auto-mana pre-loads 5 of each color.
		// City tapped for {G}. Mana Flare adds one {G}. Green = 5 + 1 + 1 = 7.
		// Blue stays at 5 (auto-mana only, no Mana Flare bonus).
		pool := g.AllPlayers()[0].ManaPool()
		green := pool.CountProducedThisTurn(core.Green)
		blue := pool.CountProducedThisTurn(core.Blue)
		if green < 7 {
			t.Errorf("expected >= 7 green (5 auto + 1 City + 1 Flare), got %d", green)
		}
		if green-blue != 2 {
			t.Errorf("Mana Flare should add exactly 1 bonus green, not other colors; green=%d blue=%d (diff should be 2)", green, blue)
		}
	})
}

// ---------------------------------------------------------------------------
// Instants/Sorceries registered in alpha_artifacts.go
// ---------------------------------------------------------------------------

func TestSacrifice(t *testing.T) {
	t.Run("sacrifice_creature_add_mana", func(t *testing.T) {
		// Sacrifice: As an additional cost to cast this spell, sacrifice a creature.
		// Add an amount of {B} equal to the sacrificed creature's mana value.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant") // CMC 4
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Sacrifice")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Sacrifice", "Hill Giant")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Hill Giant (CMC 4) sacrificed -> add {B}{B}{B}{B}.
		g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 0)
		pool := g.AllPlayers()[0].ManaPool()
		if pool.CountProducedThisTurn(core.Black) < 4 { // 4 from sacrificed Hill Giant (CMC 4)
			t.Errorf("Sacrifice should add 4 black (CMC of Hill Giant); expected >= 4 black, got %d", pool.CountProducedThisTurn(core.Black))
		}
	})
}

func TestWordOfCommand(t *testing.T) {
	t.Run("look_at_opponent_hand_and_play_card", func(t *testing.T) {
		// Word of Command: Look at target opponent's hand and choose a card from
		// it. You control that player until Word of Command finishes resolving.
		// The player plays that card if able.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Word of Command")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Word of Command", "PlayerB")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Word of Command should force PlayerB to cast Lightning Bolt.
		g.AssertHandCount(gametest.PlayerB, "Lightning Bolt", 0)
	})
}

func TestCamouflage(t *testing.T) {
	t.Run("randomizes_blocking", func(t *testing.T) {
		// Camouflage: Cast only during your declare attackers step. This turn,
		// instead of the defending player choosing blockers, you assign each
		// creature the defending player controls to block attacking creatures.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Savannah Lions")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Camouflage")
		g.CastSpell(1, core.DeclareAttackers, gametest.PlayerA, "Camouflage")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears", "Hill Giant")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Camouflage should let the attacker assign blockers.
		// At least one attacker should deal damage (5 total unblocked).
		g.AssertLife(gametest.PlayerB, 15) // 2 + 3 = 5 damage
	})
}

func TestRagingRiver(t *testing.T) {
	t.Run("splits_blockers_into_piles", func(t *testing.T) {
		// Raging River: Whenever you attack, the defending player divides non-flying
		// creatures they control into "left" and "right" piles. Each attacking
		// creature can only be blocked by creatures in the pile of your choice.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Raging River")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Gray Ogre")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		// Both creatures try to block, but Raging River restricts blocking.
		g.Block(1, gametest.PlayerB, "Hill Giant", "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Gray Ogre", "Grizzly Bears")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// With Raging River, at most one pile can block each attacker.
		// If only one of {Hill Giant, Gray Ogre} can block, Bears might get through.
		if g.AllPlayers()[1].Life() == 20 {
			t.Errorf("Raging River should restrict blocking; both blockers should not be able to block the same creature simultaneously in the same pile")
		}
	})
}

func TestNaturalSelection(t *testing.T) {
	t.Run("shuffles_library", func(t *testing.T) {
		// Natural Selection: Look at the top 3 cards of target player's library,
		// then put them back in any order. You may have that player shuffle.
		// In an automated engine, the rearrange has no effect, so we always shuffle.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Natural Selection")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Lightning Bolt")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Craw Wurm")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Natural Selection", "PlayerA")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Library should still have all 5 cards (shuffle doesn't lose cards)
		playerA := g.AllPlayers()[0]
		if len(playerA.Library()) < 5 {
			t.Errorf("Natural Selection should preserve all library cards; got %d, want >= 5", len(playerA.Library()))
		}
	})
}

func TestLich(t *testing.T) {
	t.Run("lose_life_on_entry", func(t *testing.T) {
		// Lich: As Lich enters the battlefield, you lose life equal to your life total.
		g := gametest.NewTestGame(t)
		g.SetLife(gametest.PlayerA, 20)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lich")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lich")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Life should become 0 (but you don't lose the game due to Lich).
		g.AssertLife(gametest.PlayerA, 0)
	})

	t.Run("gain_life_draws_cards", func(t *testing.T) {
		// Lich: If you would gain life, draw that many cards instead.
		g := gametest.NewTestGame(t)
		g.SetLife(gametest.PlayerA, 0)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lich")
		for range 10 {
			g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		}
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Healing Salve")
		// Cast Healing Salve to gain 3 life -> instead draw 3 cards.
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Healing Salve", "PlayerA")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Should have drawn 3 cards (not gained life).
		playerA := g.AllPlayers()[0]
		if len(playerA.Hand()) < 3 {
			t.Errorf("Lich should replace life gain with card draw; hand has %d, want >= 3", len(playerA.Hand()))
		}
		g.AssertLife(gametest.PlayerA, 0) // life should not change
	})

	t.Run("damage_sacrifices_permanents", func(t *testing.T) {
		// Lich: 3 damage -> sacrifice 3 permanents. With only Lich + 2 Plains
		// = 3 permanents available, all are sacrificed including Lich.
		// Sacrificing Lich is a battlefield -> graveyard transition (CR 700.4),
		// which fires the "When Lich is put into a graveyard, you lose the
		// game" trigger.
		g := gametest.NewTestGame(t)
		g.SetLife(gametest.PlayerA, 0)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lich")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 2)
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "PlayerA")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// All 3 permanents sacrificed; Lich's lose-game trigger fired.
		g.AssertGraveyardCount(gametest.PlayerA, "Lich", 1)
		g.AssertWinner(gametest.PlayerB)
	})

	t.Run("lose_when_lich_leaves", func(t *testing.T) {
		// Lich: If Lich is put into a graveyard, you lose the game.
		g := gametest.NewTestGame(t)
		g.SetLife(gametest.PlayerA, 0)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lich")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Disenchant")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Disenchant", "Lich")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// PlayerA should lose the game when Lich is destroyed.
		g.AssertPermanentCount(gametest.PlayerA, "Lich", 0)
	})
}

func TestIslandSanctuary(t *testing.T) {
	t.Run("skip_draw_prevents_attacks", func(t *testing.T) {
		// Island Sanctuary: If you would draw a card during your draw step, you may
		// skip that draw instead. If you do, until your next turn, you can't be
		// attacked except by creatures with flying or islandwalk.
		// Sanctuary on PlayerB so its draw-step trigger fires on turn 2; per CR
		// 103.8a PlayerA's turn-1 draw step is skipped.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island Sanctuary")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // no flying
		for range 5 {
			g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
			g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Forest")
		}
		// Turn 2 PlayerB skips draw. Turn 3 PlayerA attacks.
		g.Attack(3, gametest.PlayerA, "Grizzly Bears")
		g.StopAt(3, core.EndCombat)
		g.Execute()
		// Grizzly Bears (no flying/islandwalk) can't attack PlayerB.
		g.AssertLife(gametest.PlayerB, 20)
	})
}

func TestManaShort(t *testing.T) {
	t.Run("taps_all_lands_and_empties_pool", func(t *testing.T) {
		// Mana Short: Tap all lands target player controls and that player loses
		// all unspent mana.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Mana Short")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Mana Short", "PlayerB")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// All of PlayerB's lands should be tapped.
		g.AssertTapped(gametest.PlayerB, "Plains", true)
		g.AssertTapped(gametest.PlayerB, "Island", true)
	})
}

func TestDrainPower(t *testing.T) {
	t.Run("steals_opponent_mana", func(t *testing.T) {
		// Drain Power: Target player activates a mana ability of each land they
		// control. Then that player loses all unspent mana and you add the mana
		// lost this way.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Drain Power")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Drain Power", "PlayerB")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// PlayerB's lands tapped, PlayerA gets the mana.
		g.AssertTapped(gametest.PlayerB, "Plains", true)
		g.AssertTapped(gametest.PlayerB, "Mountain", true)
	})
}

func TestSimulacrum(t *testing.T) {
	t.Run("gain_life_and_deal_damage", func(t *testing.T) {
		// Simulacrum: You gain life equal to the damage dealt to you this turn.
		// Simulacrum deals damage to target creature you control equal to the
		// damage dealt to you this turn.
		g := gametest.NewTestGame(t)
		g.SetLife(gametest.PlayerA, 17)                                    // took 3 damage earlier this "turn"
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")    // 3/3
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // 2/2
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Simulacrum")
		// PlayerA casts Simulacrum; PlayerB bolts in response so the
		// 3 damage lands before Simulacrum resolves (CR 117.1b).
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Simulacrum", "Hill Giant")
		g.CastInResponseTo(gametest.PlayerB, "Lightning Bolt", "PlayerA")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Took 3 damage this turn -> gain 3 life (14 -> 17) and deal 3 to Hill Giant.
		g.AssertLife(gametest.PlayerA, 17)
	})
}

func TestBlazeOfGlory(t *testing.T) {
	t.Run("target_blocks_all_attackers", func(t *testing.T) {
		// Blaze of Glory: Target creature can block any number of creatures this
		// turn and must block each attacking creature if able.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gray Ogre")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Blaze of Glory")
		g.CastSpell(1, core.DeclareAttackers, gametest.PlayerB, "Blaze of Glory", "Hill Giant")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears", "Gray Ogre")
		g.Block(1, gametest.PlayerB, "Hill Giant", "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Hill Giant", "Gray Ogre")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Hill Giant blocks both attackers (2 + 2 = 4 damage, lethal to 3/3).
		// With Blaze of Glory enabling multi-block, no damage gets through.
		g.AssertLife(gametest.PlayerB, 20)
	})
}

func TestFalseOrders(t *testing.T) {
	t.Run("removes_blocker_and_reassigns", func(t *testing.T) {
		// False Orders: Cast only during combat after blockers are declared.
		// Remove target creature defending player controls from combat.
		// Attacker(s) it blocked that other creatures blocked become unblocked.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Craw Wurm")  // 6/4
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3 blocker
		g.AddCard(core.ZoneHand, gametest.PlayerA, "False Orders")
		g.Attack(1, gametest.PlayerA, "Craw Wurm")
		g.Block(1, gametest.PlayerB, "Hill Giant", "Craw Wurm")
		// Cast False Orders after blocks to remove Hill Giant from combat.
		g.CastSpell(1, core.DeclareBlockers, gametest.PlayerA, "False Orders", "Hill Giant")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Hill Giant removed from combat -> Craw Wurm becomes unblocked -> 6 damage.
		g.AssertLife(gametest.PlayerB, 14)
	})
}

func TestSirensCall(t *testing.T) {
	t.Run("forces_creatures_to_attack", func(t *testing.T) {
		// Siren's Call: Cast only during an opponent's turn, before attackers
		// are declared. Creatures the active player controls attack this turn if
		// able. At end of turn, destroy all non-Wall creatures that player
		// controls that didn't attack. Ignore this effect for each creature the
		// player didn't control continuously since the beginning of the turn.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")    // 3/3
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Siren's Call")
		// Cast on PlayerB's turn before attackers.
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerA, "Siren's Call")
		// PlayerB doesn't attack with either creature.
		g.StopAt(2, core.Cleanup)
		g.Execute()
		// Non-attackers should be destroyed at end of turn.
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 0)
	})
}

func TestMagicalHack(t *testing.T) {
	t.Run("changes_land_type_word", func(t *testing.T) {
		// Magical Hack: Change the text of target permanent by replacing all
		// instances of one basic land type with another.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Bog Wraith") // has swampwalk
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Magical Hack")
		// Change "swamp" -> "forest" on Bog Wraith.
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Magical Hack", "Bog Wraith")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Bog Wraith should have forestwalk instead of swampwalk.
		g.AssertHasAbility(gametest.PlayerA, "Bog Wraith", core.Forestwalk, true)
		g.AssertHasAbility(gametest.PlayerA, "Bog Wraith", core.Swampwalk, false)
	})
}

func TestChaosOrb(t *testing.T) {
	t.Run("destroys_random_nontoken_permanent", func(t *testing.T) {
		// Chaos Orb (simplified): {1}, {T}: Destroy a random nontoken permanent
		// you don't control. Then destroy Chaos Orb.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Chaos Orb")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Chaos Orb")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Should destroy a random permanent then destroy itself.
		g.AssertPermanentCount(gametest.PlayerA, "Chaos Orb", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 0)
	})
}

// ---------------------------------------------------------------------------
// Enchantments registered in alpha_artifacts.go
// ---------------------------------------------------------------------------

func TestPowerSurge(t *testing.T) {
	t.Run("deals_damage_for_untapped_lands", func(t *testing.T) {
		// Power Surge: At the beginning of each player's upkeep, Power Surge deals
		// X damage to that player, where X is the number of untapped lands they
		// controlled at the beginning of this turn.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Power Surge")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains", 3)
		for range 5 {
			g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
			g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Forest")
		}
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		// PlayerB had 3 untapped lands -> takes 3 damage at upkeep.
		g.AssertLife(gametest.PlayerB, 17)
	})
}

func TestLifetap(t *testing.T) {
	t.Run("gain_life_when_opponent_forest_taps", func(t *testing.T) {
		// Lifetap: Whenever a Forest an opponent controls becomes tapped,
		// you gain 1 life.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lifetap")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerB, "Forest")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		// Opponent tapped Forest -> gain 1 life.
		g.AssertLife(gametest.PlayerA, 21)
	})
}

func TestConversion(t *testing.T) {
	t.Run("mountains_become_plains", func(t *testing.T) {
		// Conversion: All Mountains are Plains.
		g := gametest.NewTestGame(t)
		// Add 2 Plains first so Conversion can pay {W}{W} at upkeep and survive.
		// Plains are added before Mountain so TryPayCostFromLands picks them first.
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Conversion")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Mountain")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Mountain should produce {W} instead of {R}.
		pool := g.AllPlayers()[0].ManaPool()
		if pool.CountProducedThisTurn(core.White) < 6 { // 5 auto + 1 from converted Mountain
			t.Errorf("Conversion should make Mountain produce {W}; expected >= 6 white, got %d", pool.CountProducedThisTurn(core.White))
		}
	})

	t.Run("sacrifice_unless_pay_WW", func(t *testing.T) {
		// At the beginning of your upkeep, sacrifice Conversion unless you pay {W}{W}.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Conversion")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// No {W}{W} paid -> Conversion sacrificed.
		g.AssertPermanentCount(gametest.PlayerA, "Conversion", 0)
	})
}

func TestGloom(t *testing.T) {
	t.Run("white_spells_cost_3_more", func(t *testing.T) {
		// Gloom: White spells cost {3} more to cast.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Gloom")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Swords to Plowshares") // {W}
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		// Swords now costs {3}{W} due to Gloom. Auto-mana adds {W}, not enough
		// for the extra {3}. If Gloom works, the spell should fail to resolve.
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Swords to Plowshares", "Hill Giant")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// With Gloom, StP can't be cast (not enough mana from auto-mana for the
		// extra {3}), so Hill Giant should survive.
		g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 1)
	})
}

func TestMagneticMountain(t *testing.T) {
	t.Run("blue_creatures_dont_untap", func(t *testing.T) {
		// Magnetic Mountain: Blue creatures don't untap during their controller's
		// untap step.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Magnetic Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Air Elemental") // blue 4/4
		g.Attack(2, gametest.PlayerB, "Air Elemental")
		// Turn 4 is PlayerB's next untap step.
		g.StopAt(4, core.PrecombatMain)
		g.Execute()
		// Air Elemental (blue) should NOT untap.
		g.AssertTapped(gametest.PlayerB, "Air Elemental", true)
	})
}

func TestConsecratedLand(t *testing.T) {
	t.Run("enchanted_land_indestructible", func(t *testing.T) {
		// Consecrate Land: Enchanted land has indestructible and can't be
		// enchanted by other Auras.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Consecrate Land")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Armageddon")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Consecrate Land", "Plains")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Armageddon")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Armageddon destroys all lands, but Consecrate Land makes Plains indestructible.
		g.AssertPermanentCount(gametest.PlayerA, "Plains", 1)
	})
}

func TestFastbond(t *testing.T) {
	t.Run("play_multiple_lands", func(t *testing.T) {
		// Fastbond: You may play any number of lands on each of your turns.
		// Whenever a land enters the battlefield under your control, if it wasn't
		// the first land you played this turn, Fastbond deals 1 damage to you.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fastbond")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Mountain")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Should be able to play all 3 lands.
		g.AssertPermanentCount(gametest.PlayerA, "Forest", 1)
		g.AssertPermanentCount(gametest.PlayerA, "Plains", 1)
		g.AssertPermanentCount(gametest.PlayerA, "Mountain", 1)
	})

	t.Run("takes_damage_after_first", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fastbond")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Mountain")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// 1st land: free. 2nd and 3rd: 1 damage each = 2 total.
		g.AssertLife(gametest.PlayerA, 18)
	})
}

func TestKudzu(t *testing.T) {
	t.Run("destroys_land_when_tapped", func(t *testing.T) {
		// Kudzu: When enchanted land becomes tapped, destroy it. If it does,
		// attach Kudzu to another land.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Kudzu")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Kudzu", "Forest")
		// Tap the enchanted Forest.
		g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerB, "Forest")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		// Forest tapped -> destroyed. Kudzu moves to Plains.
		g.AssertPermanentCount(gametest.PlayerB, "Forest", 0)
		g.AssertAttachedTo(gametest.PlayerB, "Kudzu", "Plains")
	})
}

func TestRegenerationAura(t *testing.T) {
	t.Run("activate_to_regenerate", func(t *testing.T) {
		// Regeneration (Aura): {G}: Regenerate enchanted creature.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Regeneration")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Regeneration", "Grizzly Bears")
		// Activate regeneration shield, then Bolt the creature.
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Grizzly Bears should regenerate from Bolt damage.
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}

// ---------------------------------------------------------------------------
// Auras registered in alpha_enchantments.go (land enchantments)
// ---------------------------------------------------------------------------

func TestPsychicVenom(t *testing.T) {
	t.Run("deals_2_damage_when_land_taps", func(t *testing.T) {
		// Psychic Venom: Whenever enchanted land becomes tapped, Psychic Venom
		// deals 2 damage to that land's controller.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Psychic Venom")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Psychic Venom", "Forest")
		g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerB, "Forest")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		// Forest tapped -> Psychic Venom deals 2 to PlayerB.
		g.AssertLife(gametest.PlayerB, 18)
	})
}

func TestEvilPresence(t *testing.T) {
	t.Run("land_becomes_swamp", func(t *testing.T) {
		// Evil Presence: Enchanted land is a Swamp.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Evil Presence")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Evil Presence", "Forest")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Forest should now be a Swamp (produce {B} instead of {G}).
		perm := g.FindPermanentByName("Forest", g.AllPlayers()[1].PlayerID())
		if perm == nil {
			t.Fatal("Forest not found")
		}
		if !perm.HasSubType("Swamp") {
			t.Errorf("Evil Presence should make Forest a Swamp")
		}
	})
}

func TestPhantasmalTerrain(t *testing.T) {
	t.Run("land_becomes_chosen_type", func(t *testing.T) {
		// Phantasmal Terrain: As Phantasmal Terrain enters the battlefield,
		// choose a basic land type. Enchanted land is the chosen type.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Phantasmal Terrain")
		g.ChooseManaColor(gametest.PlayerA, core.Blue) // choose Island
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Phantasmal Terrain", "Mountain")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Mountain should become an Island (Blue → Island).
		perm := g.FindPermanentByName("Mountain", g.AllPlayers()[1].PlayerID())
		if perm == nil {
			t.Fatal("Mountain not found")
		}
		if !perm.HasSubType("Island") {
			t.Errorf("Phantasmal Terrain should change Mountain to Island")
		}
	})

	t.Run("land_becomes_swamp_when_black_chosen", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Phantasmal Terrain")
		g.ChooseManaColor(gametest.PlayerA, core.Black) // choose Swamp
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Phantasmal Terrain", "Forest")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		perm := g.FindPermanentByName("Forest", g.AllPlayers()[1].PlayerID())
		if perm == nil {
			t.Fatal("Forest not found")
		}
		if !perm.HasSubType("Swamp") {
			t.Errorf("Phantasmal Terrain with Black choice should change Forest to Swamp")
		}
	})
}

func TestKormusBell(t *testing.T) {
	t.Run("swamps_become_creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kormus Bell")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		perm := g.FindPermanentByName("Swamp", g.AllPlayers()[0].PlayerID())
		if perm == nil {
			t.Fatal("Swamp not found")
		}
		if !perm.HasType(core.TypeCreature) {
			t.Errorf("Kormus Bell should make Swamp a creature")
		}
		g.AssertPowerToughness(gametest.PlayerA, "Swamp", 1, 1)
	})

	t.Run("swamps_can_attack", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kormus Bell")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.Attack(1, gametest.PlayerA, "Swamp")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 19)
	})

	t.Run("non_swamps_unaffected", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kormus Bell")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		perm := g.FindPermanentByName("Forest", g.AllPlayers()[0].PlayerID())
		if perm == nil {
			t.Fatal("Forest not found")
		}
		if perm.HasType(core.TypeCreature) {
			t.Errorf("Kormus Bell should not affect non-Swamp lands")
		}
	})

	t.Run("swamps_are_black", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kormus Bell")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		perm := g.FindPermanentByName("Swamp", g.AllPlayers()[0].PlayerID())
		if perm == nil {
			t.Fatal("Swamp not found")
		}
		hasBlack := false
		for _, col := range perm.Colors() {
			if col == core.Black {
				hasBlack = true
			}
		}
		if !hasBlack {
			t.Error("Kormus Bell should make animated Swamps black")
		}
	})
}

func TestHelmOfChatzuk(t *testing.T) {
	t.Run("grants_banding", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Helm of Chatzuk")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Helm of Chatzuk", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Banding, true)
		g.AssertTapped(gametest.PlayerA, "Helm of Chatzuk", true)
	})
}

func TestGlassesOfUrza(t *testing.T) {
	t.Run("taps_to_look", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Glasses of Urza")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Glasses of Urza", "PlayerB")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Glasses of Urza", true)
	})
}

func TestSunglassesOfUrza(t *testing.T) {
	t.Run("red_mana_as_white", func(t *testing.T) {
		// Sunglasses of Urza lets you spend red mana as white.
		// Cast Swords to Plowshares ({W}) using only red mana sources.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sunglasses of Urza")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Swords to Plowshares")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Swords to Plowshares", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Swords should exile Grizzly Bears using red mana as white.
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
	})
}

func TestJadeStatue(t *testing.T) {
	t.Run("becomes_creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Jade Statue")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Jade Statue")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		perm := g.FindPermanentByName("Jade Statue", g.AllPlayers()[0].PlayerID())
		if perm == nil {
			t.Fatal("Jade Statue not found")
		}
		if !perm.HasType(core.TypeCreature) {
			t.Errorf("Jade Statue should become a creature after activation")
		}
		g.AssertPowerToughness(gametest.PlayerA, "Jade Statue", 3, 6)
	})

	t.Run("creature_until_end_of_combat", func(t *testing.T) {
		// Jade Statue should be a creature during combat (DeclareAttackers)
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Jade Statue")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Jade Statue")
		g.StopAt(1, core.DeclareAttackers)
		g.Execute()
		perm := g.FindPermanentByName("Jade Statue", g.AllPlayers()[0].PlayerID())
		if perm == nil {
			t.Fatal("Jade Statue not found")
		}
		if !perm.HasType(core.TypeCreature) {
			t.Errorf("Jade Statue should be a creature during combat")
		}
	})

	t.Run("not_creature_after_combat", func(t *testing.T) {
		// Jade Statue should revert to non-creature after EndCombat
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Jade Statue")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Jade Statue")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		perm := g.FindPermanentByName("Jade Statue", g.AllPlayers()[0].PlayerID())
		if perm == nil {
			t.Fatal("Jade Statue not found")
		}
		if perm.HasType(core.TypeCreature) {
			t.Errorf("Jade Statue should NOT be a creature during postcombat main")
		}
	})

	t.Run("reverts_at_end_of_turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Jade Statue")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Jade Statue")
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		perm := g.FindPermanentByName("Jade Statue", g.AllPlayers()[0].PlayerID())
		if perm == nil {
			t.Fatal("Jade Statue not found")
		}
		if perm.HasType(core.TypeCreature) {
			t.Errorf("Jade Statue should revert to non-creature at end of turn")
		}
	})
}

func TestJadeMonolith(t *testing.T) {
	t.Run("redirects_creature_damage_to_controller", func(t *testing.T) {
		// {1}: The next time a source of your choice would deal damage to
		// target creature this turn, that source deals that damage to you instead.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Jade Monolith")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		// Activate Jade Monolith targeting Grizzly Bears — damage redirects to controller (PlayerA)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Jade Monolith", "Grizzly Bears")
		// Lightning Bolt targets Grizzly Bears — damage redirected to PlayerA
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Bears should survive (damage redirected), PlayerA takes 3
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
		g.AssertLife(gametest.PlayerA, 17)
	})
}

func TestPersonalIncarnation(t *testing.T) {
	t.Run("no_flying", func(t *testing.T) {
		// Oracle text does not grant Flying.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Personal Incarnation")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Personal Incarnation", core.Flying, false)
		g.AssertPowerToughness(gametest.PlayerA, "Personal Incarnation", 6, 6)
	})

	t.Run("lose_half_life_on_death", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.SetLife(gametest.PlayerA, 20)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Personal Incarnation")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		// Deal 6 damage to kill the 6/6
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Personal Incarnation")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Personal Incarnation")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Personal Incarnation", 0)
		// 20 / 2 = 10 life lost, so 10 remaining
		g.AssertLife(gametest.PlayerA, 10)
	})

	t.Run("activated_redirects_creature_damage_to_owner", func(t *testing.T) {
		// {0}: The next 1 damage that would be dealt to this creature this turn
		// is dealt to its owner instead.
		g := gametest.NewTestGame(t)
		g.SetLife(gametest.PlayerA, 20)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Personal Incarnation") // 6/6
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		// Activate the {0} ability to redirect creature damage to owner
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Personal Incarnation")
		// Bolt targets the creature — damage redirects to PlayerA
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Personal Incarnation")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Personal Incarnation should survive (damage redirected to owner)
		g.AssertPermanentCount(gametest.PlayerA, "Personal Incarnation", 1)
		// PlayerA takes the 3 damage instead
		g.AssertLife(gametest.PlayerA, 17)
	})
}

func TestVeteranBodyguard(t *testing.T) {
	t.Run("redirects_combat_damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Veteran Bodyguard") // 2/5
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")        // 3/3
		g.Attack(1, gametest.PlayerA, "Hill Giant")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Veteran Bodyguard should absorb the 3 combat damage
		g.AssertLife(gametest.PlayerB, 20)
		// Bodyguard should have taken 3 damage (5 toughness - 3 = 2 remaining)
		g.AssertPermanentCount(gametest.PlayerB, "Veteran Bodyguard", 1)
	})

	t.Run("no_redirect_when_tapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Veteran Bodyguard") // 2/5
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Icy Manipulator")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant") // 3/3
		// Tap bodyguard before combat
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerB, "Icy Manipulator", "Veteran Bodyguard")
		g.Attack(1, gametest.PlayerA, "Hill Giant")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Bodyguard is tapped, so damage goes through to player
		g.AssertLife(gametest.PlayerB, 17)
	})
}

func TestNettlingImpForceAttack(t *testing.T) {
	t.Run("forces_creature_to_attack", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Nettling Imp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3
		// Nettling Imp gives MustAttack to Hill Giant
		g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerA, "Nettling Imp", "Hill Giant")
		g.StopAt(2, core.EndCombat)
		g.Execute()
		// Hill Giant should have been forced to attack
		g.AssertTapped(gametest.PlayerA, "Nettling Imp", true)
		// Hill Giant should be tapped (it attacked)
		g.AssertTapped(gametest.PlayerB, "Hill Giant", true)
		// PlayerA should have taken 3 damage from the forced attack
		g.AssertLife(gametest.PlayerA, 17)
	})

	t.Run("cannot_target_wall", func(t *testing.T) {
		// Nettling Imp cannot target Wall creatures
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Nettling Imp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wall of Stone") // 0/8 Wall
		// Try to target a Wall — should fail; Imp won't tap
		g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerA, "Nettling Imp", "Wall of Stone")
		g.StopAt(2, core.Cleanup)
		g.Execute()
		// Nettling Imp should NOT have tapped (no valid target)
		g.AssertTapped(gametest.PlayerA, "Nettling Imp", false)
	})

	t.Run("destroy_if_didnt_attack", func(t *testing.T) {
		// If the target creature didn't attack this turn, destroy it at end of turn.
		// Use Meekstone to keep Hill Giant tapped (power >= 3 can't untap).
		// Hill Giant attacks on turn 2, gets tapped. Meekstone prevents untap.
		// On turn 4 (PlayerB's next turn), Hill Giant is still tapped.
		// Nettling Imp targets Hill Giant -> can't attack -> destroyed at EOT.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Nettling Imp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Meekstone")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3
		// Turn 2: Hill Giant attacks (forced or voluntary), gets tapped
		g.Attack(2, gametest.PlayerB, "Hill Giant")
		// Turn 4: Meekstone keeps Hill Giant tapped. Nettling Imp targets it.
		g.ActivateAbility(4, core.PrecombatMain, gametest.PlayerA, "Nettling Imp", "Hill Giant")
		g.StopAt(4, core.Cleanup)
		g.Execute()
		// Hill Giant was tapped and couldn't attack, so it should be destroyed at end of turn
		g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 0)
	})
}

func TestCyclopeanTomb(t *testing.T) {
	t.Run("turns_land_into_swamp", func(t *testing.T) {
		// {2}, {T}: Put a mire counter on target non-Swamp land.
		// As long as Cyclopean Tomb is on the battlefield, that land is a Swamp.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Cyclopean Tomb")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Cyclopean Tomb", "Forest")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		perm := g.FindPermanentByName("Forest", g.AllPlayers()[1].PlayerID())
		if perm == nil {
			t.Fatal("Forest not found")
		}
		if !perm.HasSubType("Swamp") {
			t.Errorf("Cyclopean Tomb should make Forest a Swamp")
		}
		// Forest should have a Mire counter
		if perm.Counters[core.Mire] < 1 {
			t.Errorf("Forest should have a Mire counter, got %d", perm.Counters[core.Mire])
		}
	})

	t.Run("reverts_when_tomb_leaves", func(t *testing.T) {
		// When Cyclopean Tomb leaves the battlefield, the subtype override ends.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Cyclopean Tomb")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Disenchant")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Cyclopean Tomb", "Forest")
		// Destroy Cyclopean Tomb
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Disenchant", "Cyclopean Tomb")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		perm := g.FindPermanentByName("Forest", g.AllPlayers()[1].PlayerID())
		if perm == nil {
			t.Fatal("Forest not found")
		}
		if perm.HasSubType("Swamp") {
			t.Errorf("Forest should revert to non-Swamp after Cyclopean Tomb leaves")
		}
	})

	t.Run("cannot_target_swamp", func(t *testing.T) {
		// Can only target non-Swamp lands
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Cyclopean Tomb")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Swamp")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Cyclopean Tomb", "Swamp")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Cyclopean Tomb should NOT have tapped (no valid target)
		g.AssertTapped(gametest.PlayerA, "Cyclopean Tomb", false)
	})
}

func TestIllusionaryMask(t *testing.T) {
	t.Run("enters_as_2_2_face_down", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Illusionary Mask")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Serra Angel")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Illusionary Mask", "Serra Angel")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Serra Angel should be on the battlefield as a face-down 2/2
		g.AssertPermanentCount(gametest.PlayerA, "Serra Angel", 1)
		g.AssertPowerToughness(gametest.PlayerA, "Serra Angel", 2, 2)
		perm := g.FindPermanentByName("Serra Angel", g.AllPlayers()[0].PlayerID())
		if perm == nil {
			t.Fatal("Serra Angel not found on battlefield")
		}
		if !perm.FaceDown {
			t.Error("Serra Angel should be face down")
		}
	})

	t.Run("flips_when_dealt_damage", func(t *testing.T) {
		// Put Craw Wurm (6/4) face-down via Mask, then Bolt it.
		// 3 damage flips it to 6/4 — survives (3 < 4 toughness).
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Illusionary Mask")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Craw Wurm")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Illusionary Mask", "Craw Wurm")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Craw Wurm")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Craw Wurm should have flipped face up: 6/4 with 3 damage — survives
		g.AssertPermanentCount(gametest.PlayerA, "Craw Wurm", 1)
		perm := g.FindPermanentByName("Craw Wurm", g.AllPlayers()[0].PlayerID())
		if perm == nil {
			t.Fatal("Craw Wurm not found on battlefield")
		}
		if perm.FaceDown {
			t.Error("Craw Wurm should be face up after taking damage")
		}
		g.AssertPowerToughness(gametest.PlayerA, "Craw Wurm", 6, 4)
	})

	t.Run("dies_if_flip_reveals_lethal", func(t *testing.T) {
		// Put Hill Giant (3/3) face-down. Bolt it (3 damage).
		// Flips to 3/3 with 3 damage — lethal, SBA destroys it.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Illusionary Mask")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Illusionary Mask", "Hill Giant")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Hill Giant")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Hill Giant should be dead: 3/3 with 3 damage
		g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 0)
		g.AssertGraveyardCount(gametest.PlayerA, "Hill Giant", 1)
	})

	t.Run("no_abilities_while_face_down", func(t *testing.T) {
		// Put Serra Angel (flying, vigilance) face-down.
		// Verify it does NOT have Flying while face-down.
		// Then deal 1 damage (Prodigal Sorcerer) to flip it.
		// After flip, verify it HAS Flying.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Illusionary Mask")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Serra Angel")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Prodigal Sorcerer")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Illusionary Mask", "Serra Angel")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Face-down: no Flying
		g.AssertHasAbility(gametest.PlayerA, "Serra Angel", core.Flying, false)
		// Now deal 1 damage with Prodigal Sorcerer to flip it
		g.ActivateAbility(1, core.BeginCombat, gametest.PlayerB, "Prodigal Sorcerer", "Serra Angel")
		g.StopAt(1, core.DeclareAttackers)
		g.Execute()
		// Face-up: has Flying
		g.AssertHasAbility(gametest.PlayerA, "Serra Angel", core.Flying, true)
	})
}

func TestInstillEnergy(t *testing.T) {
	t.Run("grants_haste_and_untap", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Instill Energy")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Instill Energy", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		// Untap after combat via granted ability
		g.ActivateAbility(1, core.PostcombatMain, gametest.PlayerA, "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Grizzly Bears", false)
		g.AssertLife(gametest.PlayerB, 18)
	})

	t.Run("untap_limited_to_once_per_turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Instill Energy")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		// Also give PlayerA an Icy Manipulator to tap the bears a second time
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Icy Manipulator")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Instill Energy", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		// First untap — should work
		g.ActivateAbility(1, core.PostcombatMain, gametest.PlayerA, "Grizzly Bears")
		// Tap bears again with Icy Manipulator
		g.ActivateAbility(1, core.PostcombatMain, gametest.PlayerA, "Icy Manipulator", "Grizzly Bears")
		// Second untap attempt — should fail (once per turn)
		g.ActivateAbility(1, core.PostcombatMain, gametest.PlayerA, "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Bears should remain tapped
		g.AssertTapped(gametest.PlayerA, "Grizzly Bears", true)
	})
}
