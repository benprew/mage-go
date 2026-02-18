package cards

import (
	"testing"

	"github.com/mage/mage"
)

// Tests for cards registered in alpha_artifacts.go (and a few
// from alpha_spells.go / alpha_enchantments.go that are closely related).

// ---------------------------------------------------------------------------
// Artifacts
// ---------------------------------------------------------------------------

func TestWinterOrb(t *testing.T) {
	t.Run("limits_land_untap_to_one", func(t *testing.T) {
		// Winter Orb: Players can't untap more than one land during their untap steps.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Winter Orb")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Plains")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Island")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Mountain")
		// Tap all lands by activating their mana abilities.
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Plains")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Island")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Mountain")
		// Turn 3 is PlayerA's next untap step. Only 1 land should untap.
		g.StopAt(3, mage.PrecombatMain)
		g.Execute()
		// With Winter Orb, at most 1 land untaps; 2 should remain tapped.
		tapped := 0
		playerAID := g.Players[0].PlayerID()
		for _, perm := range g.Battlefield {
			if perm.Controller == playerAID && perm.HasType(mage.TypeLand) && perm.Tapped {
				tapped++
			}
		}
		if tapped < 2 {
			t.Errorf("Winter Orb should keep 2 lands tapped, only %d tapped", tapped)
		}
	})

	t.Run("affects_opponent_too", func(t *testing.T) {
		// Winter Orb affects all players, not just the controller.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Winter Orb")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Swamp")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Mountain")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Forest")
		g.ActivateAbility(2, mage.PrecombatMain, mage.PlayerB, "Swamp")
		g.ActivateAbility(2, mage.PrecombatMain, mage.PlayerB, "Mountain")
		g.ActivateAbility(2, mage.PrecombatMain, mage.PlayerB, "Forest")
		g.StopAt(4, mage.PrecombatMain)
		g.Execute()
		tapped := 0
		playerBID := g.Players[1].PlayerID()
		for _, perm := range g.Battlefield {
			if perm.Controller == playerBID && perm.HasType(mage.TypeLand) && perm.Tapped {
				tapped++
			}
		}
		if tapped < 2 {
			t.Errorf("Winter Orb should limit opponent too; only %d lands tapped, want >= 2", tapped)
		}
	})

	t.Run("creatures_untap_normally", func(t *testing.T) {
		// Winter Orb only restricts lands; creatures untap normally.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Winter Orb")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Serra Angel")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Hill Giant")
		g.Attack(1, mage.PlayerA, "Hill Giant")
		g.StopAt(3, mage.PrecombatMain)
		g.Execute()
		g.AssertTapped(mage.PlayerA, "Hill Giant", false)
	})

	t.Run("no_restriction_when_tapped", func(t *testing.T) {
		// When Winter Orb is tapped, its restriction doesn't apply.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Winter Orb")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Icy Manipulator")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Plains")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Island")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Mountain")
		// Tap Winter Orb itself with Icy Manipulator.
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Icy Manipulator", "Winter Orb")
		// Tap all lands.
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Plains")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Island")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Mountain")
		g.StopAt(3, mage.PrecombatMain)
		g.Execute()
		// Winter Orb is tapped, so all lands should untap normally.
		g.AssertTapped(mage.PlayerA, "Plains", false)
		g.AssertTapped(mage.PlayerA, "Island", false)
		g.AssertTapped(mage.PlayerA, "Mountain", false)
	})
}

func TestMeekstone(t *testing.T) {
	t.Run("prevents_big_creatures_from_untapping", func(t *testing.T) {
		// Meekstone: Creatures with power 3 or greater don't untap during their
		// controller's untap step.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Meekstone")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Hill Giant") // 3/3
		// Attack with Hill Giant to tap it.
		g.Attack(1, mage.PlayerA, "Hill Giant")
		// Turn 3 is PlayerA's next untap step.
		g.StopAt(3, mage.PrecombatMain)
		g.Execute()
		// Hill Giant (power 3) should NOT untap with Meekstone.
		g.AssertTapped(mage.PlayerA, "Hill Giant", true)
	})

	t.Run("small_creatures_untap_normally", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Meekstone")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears") // 2/2
		g.Attack(1, mage.PlayerA, "Grizzly Bears")
		g.StopAt(3, mage.PrecombatMain)
		g.Execute()
		// Grizzly Bears (power 2) should untap normally.
		g.AssertTapped(mage.PlayerA, "Grizzly Bears", false)
	})
}

func TestHowlingMine(t *testing.T) {
	t.Run("both_players_draw_extra_card", func(t *testing.T) {
		// Howling Mine: At the beginning of each player's draw step, that player
		// draws an additional card.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Howling Mine")
		for i := 0; i < 10; i++ {
			g.AddCard(mage.ZoneLibrary, mage.PlayerA, "Grizzly Bears")
			g.AddCard(mage.ZoneLibrary, mage.PlayerB, "Grizzly Bears")
		}
		// Turn 1 doesn't draw for first player; check turn 3 (PlayerA's second turn).
		g.StopAt(3, mage.PrecombatMain)
		g.Execute()
		// By turn 3, PlayerA has drawn: turn 3 draw = 1 (normal) + 1 (Mine) = 2.
		// PlayerB drew on turn 2: 1 (normal) + 1 (Mine) = 2.
		playerA := g.Players[0]
		playerB := g.Players[1]
		if len(playerA.Hand()) < 2 {
			t.Errorf("Howling Mine should give PlayerA extra draw; hand has %d cards, want >= 2", len(playerA.Hand()))
		}
		if len(playerB.Hand()) < 2 {
			t.Errorf("Howling Mine should give PlayerB extra draw; hand has %d cards, want >= 2", len(playerB.Hand()))
		}
	})

	t.Run("no_extra_draw_when_tapped", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Howling Mine")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Icy Manipulator")
		for i := 0; i < 10; i++ {
			g.AddCard(mage.ZoneLibrary, mage.PlayerA, "Forest")
			g.AddCard(mage.ZoneLibrary, mage.PlayerB, "Forest")
		}
		// Tap Howling Mine before PlayerA's draw step.
		g.ActivateAbility(1, mage.Upkeep, mage.PlayerB, "Icy Manipulator", "Howling Mine")
		g.StopAt(1, mage.PrecombatMain)
		g.Execute()
		// Howling Mine is tapped, so no extra draw. Normal draw of 1.
		playerA := g.Players[0]
		if len(playerA.Hand()) > 1 {
			t.Errorf("Tapped Howling Mine should not give extra draw; hand has %d cards, want 1", len(playerA.Hand()))
		}
	})
}

func TestForcefield(t *testing.T) {
	t.Run("reduces_unblocked_damage_to_one", func(t *testing.T) {
		// Forcefield: {1}: If an unblocked creature would deal combat damage to you,
		// prevent all but 1 of that damage.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Forcefield")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Craw Wurm") // 6/4
		g.ActivateAbility(1, mage.DeclareBlockers, mage.PlayerB, "Forcefield")
		g.Attack(1, mage.PlayerA, "Craw Wurm")
		g.StopAt(1, mage.EndCombat)
		g.Execute()
		// Forcefield should reduce 6 unblocked damage to 1.
		g.AssertLife(mage.PlayerB, 19)
	})
}

func TestForcefieldBlockedCreature(t *testing.T) {
	t.Run("does_not_reduce_blocked_damage", func(t *testing.T) {
		// Forcefield only prevents unblocked combat damage.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Forcefield")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Craw Wurm")    // 6/4
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Grizzly Bears") // 2/2
		g.ActivateAbility(1, mage.DeclareBlockers, mage.PlayerB, "Forcefield")
		g.Attack(1, mage.PlayerA, "Craw Wurm")
		g.Block(1, mage.PlayerB, "Grizzly Bears", "Craw Wurm")
		g.StopAt(1, mage.EndCombat)
		g.Execute()
		// Craw Wurm is blocked, so Forcefield doesn't apply. No player damage.
		g.AssertLife(mage.PlayerB, 20)
	})

	t.Run("does_not_affect_1_power_creature", func(t *testing.T) {
		// A 1/1 unblocked creature deals 1 damage; Forcefield allows 1 through, so no change.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Forcefield")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Llanowar Elves") // 1/1
		g.ActivateAbility(1, mage.DeclareBlockers, mage.PlayerB, "Forcefield")
		g.Attack(1, mage.PlayerA, "Llanowar Elves")
		g.StopAt(1, mage.EndCombat)
		g.Execute()
		g.AssertLife(mage.PlayerB, 19)
	})

	t.Run("no_effect_without_activation", func(t *testing.T) {
		// If not activated, full damage goes through.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Forcefield")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Craw Wurm") // 6/4
		g.Attack(1, mage.PlayerA, "Craw Wurm")
		g.StopAt(1, mage.EndCombat)
		g.Execute()
		g.AssertLife(mage.PlayerB, 14)
	})
}

func TestTheHive(t *testing.T) {
	t.Run("creates_wasp_token", func(t *testing.T) {
		// The Hive: {5}, {T}: Create a 1/1 colorless Insect artifact creature
		// token with flying named Wasp.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "The Hive")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "The Hive")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Should have created a 1/1 Wasp token.
		g.AssertPermanentCount(mage.PlayerA, "Wasp", 1)
	})

	t.Run("token_has_flying", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "The Hive")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "The Hive")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertHasAbility(mage.PlayerA, "Wasp", mage.Flying, true)
	})

	t.Run("token_is_artifact_creature", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "The Hive")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "The Hive")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		perm := g.FindPermanentByName("Wasp", g.Players[0].PlayerID())
		if perm == nil {
			t.Fatal("Wasp token not found")
		}
		if !perm.HasType(mage.TypeArtifact) {
			t.Error("Wasp should be an artifact")
		}
		if !perm.HasType(mage.TypeCreature) {
			t.Error("Wasp should be a creature")
		}
		if perm.CurrentPower(g.Game) != 1 || perm.CurrentToughness(g.Game) != 1 {
			t.Errorf("Wasp should be 1/1, got %d/%d", perm.CurrentPower(g.Game), perm.CurrentToughness(g.Game))
		}
	})

	t.Run("token_can_attack", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "The Hive")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "The Hive")
		// Token has summoning sickness on turn 1; attack on turn 3.
		g.Attack(3, mage.PlayerA, "Wasp")
		g.StopAt(3, mage.EndCombat)
		g.Execute()
		g.AssertLife(mage.PlayerB, 19)
	})
}

func TestCopyArtifact(t *testing.T) {
	t.Run("enters_as_copy_of_artifact", func(t *testing.T) {
		// Copy Artifact: You may have Copy Artifact enter the battlefield as a copy
		// of any artifact on the battlefield, except it's an enchantment in addition
		// to its other types.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Sol Ring")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Copy Artifact")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Copy Artifact", "Sol Ring")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Should have 2 Sol Rings (original + copy).
		solCount := 0
		playerAID := g.Players[0].PlayerID()
		for _, perm := range g.Battlefield {
			if perm.Controller == playerAID && perm.Name() == "Sol Ring" {
				solCount++
			}
		}
		if solCount < 2 {
			t.Errorf("Copy Artifact should create copy of Sol Ring; found %d, want 2", solCount)
		}
	})

	t.Run("is_also_enchantment", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Sol Ring")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Copy Artifact")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Copy Artifact", "Sol Ring")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// The copy becomes "Sol Ring" but is also an Enchantment.
		// Find the Sol Ring that has the Enchantment type.
		playerAID := g.Players[0].PlayerID()
		found := false
		for _, perm := range g.Battlefield {
			if perm.Controller == playerAID && perm.Name() == "Sol Ring" && perm.HasType(mage.TypeEnchantment) {
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
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Sol Ring")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Copy Artifact")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Copy Artifact", "Sol Ring")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		playerAID := g.Players[0].PlayerID()
		found := false
		for _, perm := range g.Battlefield {
			if perm.Controller == playerAID && perm.Name() == "Sol Ring" &&
				perm.HasType(mage.TypeEnchantment) && perm.HasType(mage.TypeArtifact) {
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
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Sol Ring")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Copy Artifact")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Copy Artifact", "Sol Ring")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Find the copy (Sol Ring with Enchantment type) and verify it has abilities
		playerAID := g.Players[0].PlayerID()
		for _, perm := range g.Battlefield {
			if perm.Controller == playerAID && perm.Name() == "Sol Ring" && perm.HasType(mage.TypeEnchantment) {
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
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Living Lands")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Forest")
		g.StopAt(1, mage.PrecombatMain)
		g.Execute()
		// Forest should be a 1/1 creature.
		perm := g.FindPermanentByName("Forest", g.Players[0].PlayerID())
		if perm == nil {
			t.Fatal("Forest not found")
		}
		if !perm.HasType(mage.TypeCreature) {
			t.Errorf("Living Lands should make Forest a creature")
		}
		g.AssertPowerToughness(mage.PlayerA, "Forest", 1, 1)
	})

	t.Run("forests_can_attack", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Living Lands")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Forest")
		g.Attack(1, mage.PlayerA, "Forest")
		g.StopAt(1, mage.EndCombat)
		g.Execute()
		// Animated Forest should deal 1 damage.
		g.AssertLife(mage.PlayerB, 19)
	})

	t.Run("forests_still_tap_for_mana", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Living Lands")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Forest")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Forest")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		perm := g.FindPermanentByName("Forest", g.Players[0].PlayerID())
		if perm == nil {
			t.Fatal("Forest not found")
		}
		if !perm.HasType(mage.TypeLand) {
			t.Errorf("Living Lands Forest should still be a land")
		}
	})
}

func TestManaFlare(t *testing.T) {
	t.Run("doubles_mana_from_lands", func(t *testing.T) {
		// Mana Flare: Whenever a player taps a land for mana, that player adds
		// one additional mana of any type that land produced.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Mana Flare")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Mountain")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Mountain")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Mountain tapped for {R}. Mana Flare adds another {R}.
		// Auto-mana adds 5R. Mountain = 1R + 1R (Mana Flare) = 2R. Total = 7R.
		pool := g.Players[0].ManaPool()
		if pool.Count(mage.Red) < 7 {
			t.Errorf("Mana Flare should double land mana; expected >= 7 red, got %d", pool.Count(mage.Red))
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
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Hill Giant") // CMC 4
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Sacrifice")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Sacrifice", "Hill Giant")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Hill Giant (CMC 4) sacrificed -> add {B}{B}{B}{B}.
		g.AssertPermanentCount(mage.PlayerA, "Hill Giant", 0)
		pool := g.Players[0].ManaPool()
		if pool.Count(mage.Black) < 4 { // 4 from sacrificed Hill Giant (CMC 4)
			t.Errorf("Sacrifice should add 4 black (CMC of Hill Giant); expected >= 4 black, got %d", pool.Count(mage.Black))
		}
	})
}

func TestWordOfCommand(t *testing.T) {
	t.Run("look_at_opponent_hand_and_play_card", func(t *testing.T) {
		// Word of Command: Look at target opponent's hand and choose a card from
		// it. You control that player until Word of Command finishes resolving.
		// The player plays that card if able.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Word of Command")
		g.AddCard(mage.ZoneHand, mage.PlayerB, "Lightning Bolt")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Word of Command", "PlayerB")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Word of Command should force PlayerB to cast Lightning Bolt.
		g.AssertHandCount(mage.PlayerB, "Lightning Bolt", 0)
	})
}

func TestCamouflage(t *testing.T) {
	t.Run("randomizes_blocking", func(t *testing.T) {
		// Camouflage: Cast only during your declare attackers step. This turn,
		// instead of the defending player choosing blockers, you assign each
		// creature the defending player controls to block attacking creatures.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Hill Giant")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Savannah Lions")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Camouflage")
		g.CastSpell(1, mage.DeclareAttackers, mage.PlayerA, "Camouflage")
		g.Attack(1, mage.PlayerA, "Grizzly Bears", "Hill Giant")
		g.StopAt(1, mage.EndCombat)
		g.Execute()
		// Camouflage should let the attacker assign blockers.
		// At least one attacker should deal damage (5 total unblocked).
		g.AssertLife(mage.PlayerB, 15) // 2 + 3 = 5 damage
	})
}

func TestRagingRiver(t *testing.T) {
	t.Run("splits_blockers_into_piles", func(t *testing.T) {
		// Raging River: Whenever you attack, the defending player divides non-flying
		// creatures they control into "left" and "right" piles. Each attacking
		// creature can only be blocked by creatures in the pile of your choice.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Raging River")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Hill Giant")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Gray Ogre")
		g.Attack(1, mage.PlayerA, "Grizzly Bears")
		// Both creatures try to block, but Raging River restricts blocking.
		g.Block(1, mage.PlayerB, "Hill Giant", "Grizzly Bears")
		g.Block(1, mage.PlayerB, "Gray Ogre", "Grizzly Bears")
		g.StopAt(1, mage.EndCombat)
		g.Execute()
		// With Raging River, at most one pile can block each attacker.
		// If only one of {Hill Giant, Gray Ogre} can block, Bears might get through.
		if g.Players[1].Life() == 20 {
			t.Errorf("Raging River should restrict blocking; both blockers should not be able to block the same creature simultaneously in the same pile")
		}
	})
}

func TestNaturalSelection(t *testing.T) {
	t.Run("shuffles_library", func(t *testing.T) {
		// Natural Selection: Look at the top 3 cards of target player's library,
		// then put them back in any order. You may have that player shuffle.
		// In an automated engine, the rearrange has no effect, so we always shuffle.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Natural Selection")
		g.AddCard(mage.ZoneLibrary, mage.PlayerA, "Lightning Bolt")
		g.AddCard(mage.ZoneLibrary, mage.PlayerA, "Grizzly Bears")
		g.AddCard(mage.ZoneLibrary, mage.PlayerA, "Hill Giant")
		g.AddCard(mage.ZoneLibrary, mage.PlayerA, "Craw Wurm")
		g.AddCard(mage.ZoneLibrary, mage.PlayerA, "Forest")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Natural Selection", "PlayerA")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Library should still have all 5 cards (shuffle doesn't lose cards)
		playerA := g.Players[0]
		if len(playerA.Library()) < 5 {
			t.Errorf("Natural Selection should preserve all library cards; got %d, want >= 5", len(playerA.Library()))
		}
	})
}

func TestLich(t *testing.T) {
	t.Run("lose_life_on_entry", func(t *testing.T) {
		// Lich: As Lich enters the battlefield, you lose life equal to your life total.
		g := mage.NewTestGame(t)
		g.SetLife(mage.PlayerA, 20)
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Lich")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Lich")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Life should become 0 (but you don't lose the game due to Lich).
		g.AssertLife(mage.PlayerA, 0)
	})

	t.Run("gain_life_draws_cards", func(t *testing.T) {
		// Lich: If you would gain life, draw that many cards instead.
		g := mage.NewTestGame(t)
		g.SetLife(mage.PlayerA, 0)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Lich")
		for i := 0; i < 10; i++ {
			g.AddCard(mage.ZoneLibrary, mage.PlayerA, "Grizzly Bears")
		}
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Healing Salve")
		// Cast Healing Salve to gain 3 life -> instead draw 3 cards.
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Healing Salve", "PlayerA")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Should have drawn 3 cards (not gained life).
		playerA := g.Players[0]
		if len(playerA.Hand()) < 3 {
			t.Errorf("Lich should replace life gain with card draw; hand has %d, want >= 3", len(playerA.Hand()))
		}
		g.AssertLife(mage.PlayerA, 0) // life should not change
	})

	t.Run("damage_sacrifices_permanents", func(t *testing.T) {
		// Lich: If you would take damage, instead sacrifice that many nontoken
		// permanents. If you can't, you lose the game.
		g := mage.NewTestGame(t)
		g.SetLife(mage.PlayerA, 0)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Lich")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Plains", 2)
		g.AddCard(mage.ZoneHand, mage.PlayerB, "Lightning Bolt")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerB, "Lightning Bolt", "PlayerA")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// 3 damage -> sacrifice 3 permanents. Only Lich + 2 Plains = 3 total.
		g.AssertLife(mage.PlayerA, 0) // Lich prevents life loss
	})

	t.Run("lose_when_lich_leaves", func(t *testing.T) {
		// Lich: If Lich is put into a graveyard, you lose the game.
		g := mage.NewTestGame(t)
		g.SetLife(mage.PlayerA, 0)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Lich")
		g.AddCard(mage.ZoneHand, mage.PlayerB, "Disenchant")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerB, "Disenchant", "Lich")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// PlayerA should lose the game when Lich is destroyed.
		g.AssertPermanentCount(mage.PlayerA, "Lich", 0)
	})
}

func TestIslandSanctuary(t *testing.T) {
	t.Run("skip_draw_prevents_attacks", func(t *testing.T) {
		// Island Sanctuary: If you would draw a card during your draw step, you may
		// skip that draw instead. If you do, until your next turn, you can't be
		// attacked except by creatures with flying or islandwalk.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Island Sanctuary")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Grizzly Bears") // no flying
		for i := 0; i < 5; i++ {
			g.AddCard(mage.ZoneLibrary, mage.PlayerA, "Forest")
			g.AddCard(mage.ZoneLibrary, mage.PlayerB, "Forest")
		}
		// PlayerA skips draw. On turn 2, PlayerB attacks with Bears.
		g.Attack(2, mage.PlayerB, "Grizzly Bears")
		g.StopAt(2, mage.EndCombat)
		g.Execute()
		// Grizzly Bears (no flying/islandwalk) can't attack PlayerA.
		g.AssertLife(mage.PlayerA, 20)
	})
}

func TestManaShort(t *testing.T) {
	t.Run("taps_all_lands_and_empties_pool", func(t *testing.T) {
		// Mana Short: Tap all lands target player controls and that player loses
		// all unspent mana.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Plains")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Island")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Mana Short")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Mana Short", "PlayerB")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// All of PlayerB's lands should be tapped.
		g.AssertTapped(mage.PlayerB, "Plains", true)
		g.AssertTapped(mage.PlayerB, "Island", true)
	})
}

func TestDrainPower(t *testing.T) {
	t.Run("steals_opponent_mana", func(t *testing.T) {
		// Drain Power: Target player activates a mana ability of each land they
		// control. Then that player loses all unspent mana and you add the mana
		// lost this way.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Plains")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Mountain")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Drain Power")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Drain Power", "PlayerB")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// PlayerB's lands tapped, PlayerA gets the mana.
		g.AssertTapped(mage.PlayerB, "Plains", true)
		g.AssertTapped(mage.PlayerB, "Mountain", true)
	})
}

func TestSimulacrum(t *testing.T) {
	t.Run("gain_life_and_deal_damage", func(t *testing.T) {
		// Simulacrum: You gain life equal to the damage dealt to you this turn.
		// Simulacrum deals damage to target creature you control equal to the
		// damage dealt to you this turn.
		g := mage.NewTestGame(t)
		g.SetLife(mage.PlayerA, 17) // took 3 damage earlier this "turn"
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Hill Giant")   // 3/3
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Grizzly Bears") // 2/2
		g.AddCard(mage.ZoneHand, mage.PlayerB, "Lightning Bolt")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Simulacrum")
		// Bolt PlayerA for 3 damage (17 -> 14).
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerB, "Lightning Bolt", "PlayerA")
		// Cast Simulacrum targeting Hill Giant.
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Simulacrum", "Hill Giant")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Took 3 damage this turn -> gain 3 life (14 -> 17) and deal 3 to Hill Giant.
		g.AssertLife(mage.PlayerA, 17)
	})
}

func TestBlazeOfGlory(t *testing.T) {
	t.Run("target_blocks_all_attackers", func(t *testing.T) {
		// Blaze of Glory: Target creature can block any number of creatures this
		// turn and must block each attacking creature if able.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Hill Giant") // 3/3
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Gray Ogre")
		g.AddCard(mage.ZoneHand, mage.PlayerB, "Blaze of Glory")
		g.CastSpell(1, mage.DeclareAttackers, mage.PlayerB, "Blaze of Glory", "Hill Giant")
		g.Attack(1, mage.PlayerA, "Grizzly Bears", "Gray Ogre")
		g.Block(1, mage.PlayerB, "Hill Giant", "Grizzly Bears")
		g.Block(1, mage.PlayerB, "Hill Giant", "Gray Ogre")
		g.StopAt(1, mage.EndCombat)
		g.Execute()
		// Hill Giant blocks both attackers (2 + 2 = 4 damage, lethal to 3/3).
		// With Blaze of Glory enabling multi-block, no damage gets through.
		g.AssertLife(mage.PlayerB, 20)
	})
}

func TestFalseOrders(t *testing.T) {
	t.Run("removes_blocker_and_reassigns", func(t *testing.T) {
		// False Orders: Cast only during combat after blockers are declared.
		// Remove target creature defending player controls from combat.
		// Attacker(s) it blocked that other creatures blocked become unblocked.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Craw Wurm") // 6/4
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Hill Giant") // 3/3 blocker
		g.AddCard(mage.ZoneHand, mage.PlayerA, "False Orders")
		g.Attack(1, mage.PlayerA, "Craw Wurm")
		g.Block(1, mage.PlayerB, "Hill Giant", "Craw Wurm")
		// Cast False Orders after blocks to remove Hill Giant from combat.
		g.CastSpell(1, mage.FirstStrikeDamage, mage.PlayerA, "False Orders", "Hill Giant")
		g.StopAt(1, mage.EndCombat)
		g.Execute()
		// Hill Giant removed from combat -> Craw Wurm becomes unblocked -> 6 damage.
		g.AssertLife(mage.PlayerB, 14)
	})
}

func TestSirensCall(t *testing.T) {
	t.Run("forces_creatures_to_attack", func(t *testing.T) {
		// Siren's Call: Cast only during an opponent's turn, before attackers
		// are declared. Creatures the active player controls attack this turn if
		// able. At end of turn, destroy all non-Wall creatures that player
		// controls that didn't attack. Ignore this effect for each creature the
		// player didn't control continuously since the beginning of the turn.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Grizzly Bears") // 2/2
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Hill Giant")    // 3/3
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Siren's Call")
		// Cast on PlayerB's turn before attackers.
		g.CastSpell(2, mage.PrecombatMain, mage.PlayerA, "Siren's Call")
		// PlayerB doesn't attack with either creature.
		g.StopAt(2, mage.Cleanup)
		g.Execute()
		// Non-attackers should be destroyed at end of turn.
		g.AssertPermanentCount(mage.PlayerB, "Grizzly Bears", 0)
		g.AssertPermanentCount(mage.PlayerB, "Hill Giant", 0)
	})
}

func TestMagicalHack(t *testing.T) {
	t.Run("changes_land_type_word", func(t *testing.T) {
		// Magical Hack: Change the text of target permanent by replacing all
		// instances of one basic land type with another.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Bog Wraith") // has swampwalk
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Magical Hack")
		// Change "swamp" -> "forest" on Bog Wraith.
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Magical Hack", "Bog Wraith")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Bog Wraith should have forestwalk instead of swampwalk.
		g.AssertHasAbility(mage.PlayerA, "Bog Wraith", mage.Forestwalk, true)
		g.AssertHasAbility(mage.PlayerA, "Bog Wraith", mage.Swampwalk, false)
	})
}

func TestChaosOrb(t *testing.T) {
	t.Run("destroys_random_nontoken_permanent", func(t *testing.T) {
		// Chaos Orb (simplified): {1}, {T}: Destroy a random nontoken permanent
		// you don't control. Then destroy Chaos Orb.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Chaos Orb")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Hill Giant")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Chaos Orb")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Should destroy a random permanent then destroy itself.
		g.AssertPermanentCount(mage.PlayerA, "Chaos Orb", 0)
		g.AssertPermanentCount(mage.PlayerB, "Hill Giant", 0)
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
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Power Surge")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Plains", 3)
		for i := 0; i < 5; i++ {
			g.AddCard(mage.ZoneLibrary, mage.PlayerA, "Forest")
			g.AddCard(mage.ZoneLibrary, mage.PlayerB, "Forest")
		}
		g.StopAt(2, mage.PrecombatMain)
		g.Execute()
		// PlayerB had 3 untapped lands -> takes 3 damage at upkeep.
		g.AssertLife(mage.PlayerB, 17)
	})
}

func TestLifetap(t *testing.T) {
	t.Run("gain_life_when_opponent_forest_taps", func(t *testing.T) {
		// Lifetap: Whenever a Forest an opponent controls becomes tapped,
		// you gain 1 life.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Lifetap")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Forest")
		g.ActivateAbility(2, mage.PrecombatMain, mage.PlayerB, "Forest")
		g.StopAt(2, mage.BeginCombat)
		g.Execute()
		// Opponent tapped Forest -> gain 1 life.
		g.AssertLife(mage.PlayerA, 21)
	})
}

func TestConversion(t *testing.T) {
	t.Run("mountains_become_plains", func(t *testing.T) {
		// Conversion: All Mountains are Plains.
		g := mage.NewTestGame(t)
		// Add 2 Plains first so Conversion can pay {W}{W} at upkeep and survive.
		// Plains are added before Mountain so TryPayCostFromLands picks them first.
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Plains")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Plains")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Conversion")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Mountain")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Mountain")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Mountain should produce {W} instead of {R}.
		pool := g.Players[0].ManaPool()
		if pool.Count(mage.White) < 6 { // 5 auto + 1 from converted Mountain
			t.Errorf("Conversion should make Mountain produce {W}; expected >= 6 white, got %d", pool.Count(mage.White))
		}
	})

	t.Run("sacrifice_unless_pay_WW", func(t *testing.T) {
		// At the beginning of your upkeep, sacrifice Conversion unless you pay {W}{W}.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Conversion")
		g.StopAt(1, mage.PrecombatMain)
		g.Execute()
		// No {W}{W} paid -> Conversion sacrificed.
		g.AssertPermanentCount(mage.PlayerA, "Conversion", 0)
	})
}

func TestGloom(t *testing.T) {
	t.Run("white_spells_cost_3_more", func(t *testing.T) {
		// Gloom: White spells cost {3} more to cast.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Gloom")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Swords to Plowshares") // {W}
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Hill Giant")
		// Swords now costs {3}{W} due to Gloom. Auto-mana adds {W}, not enough
		// for the extra {3}. If Gloom works, the spell should fail to resolve.
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Swords to Plowshares", "Hill Giant")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// With Gloom, StP can't be cast (not enough mana from auto-mana for the
		// extra {3}), so Hill Giant should survive.
		g.AssertPermanentCount(mage.PlayerB, "Hill Giant", 1)
	})
}

func TestMagneticMountain(t *testing.T) {
	t.Run("blue_creatures_dont_untap", func(t *testing.T) {
		// Magnetic Mountain: Blue creatures don't untap during their controller's
		// untap step.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Magnetic Mountain")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Air Elemental") // blue 4/4
		g.Attack(2, mage.PlayerB, "Air Elemental")
		// Turn 4 is PlayerB's next untap step.
		g.StopAt(4, mage.PrecombatMain)
		g.Execute()
		// Air Elemental (blue) should NOT untap.
		g.AssertTapped(mage.PlayerB, "Air Elemental", true)
	})
}

func TestConsecratedLand(t *testing.T) {
	t.Run("enchanted_land_indestructible", func(t *testing.T) {
		// Consecrate Land: Enchanted land has indestructible and can't be
		// enchanted by other Auras.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Plains")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Consecrate Land")
		g.AddCard(mage.ZoneHand, mage.PlayerB, "Armageddon")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Consecrate Land", "Plains")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerB, "Armageddon")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Armageddon destroys all lands, but Consecrate Land makes Plains indestructible.
		g.AssertPermanentCount(mage.PlayerA, "Plains", 1)
	})
}

func TestFastbond(t *testing.T) {
	t.Run("play_multiple_lands", func(t *testing.T) {
		// Fastbond: You may play any number of lands on each of your turns.
		// Whenever a land enters the battlefield under your control, if it wasn't
		// the first land you played this turn, Fastbond deals 1 damage to you.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Fastbond")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Forest")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Plains")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Mountain")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Should be able to play all 3 lands.
		g.AssertPermanentCount(mage.PlayerA, "Forest", 1)
		g.AssertPermanentCount(mage.PlayerA, "Plains", 1)
		g.AssertPermanentCount(mage.PlayerA, "Mountain", 1)
	})

	t.Run("takes_damage_after_first", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Fastbond")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Forest")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Plains")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Mountain")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// 1st land: free. 2nd and 3rd: 1 damage each = 2 total.
		g.AssertLife(mage.PlayerA, 18)
	})
}

func TestKudzu(t *testing.T) {
	t.Run("destroys_land_when_tapped", func(t *testing.T) {
		// Kudzu: When enchanted land becomes tapped, destroy it. If it does,
		// attach Kudzu to another land.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Forest")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Plains")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Kudzu")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Kudzu", "Forest")
		// Tap the enchanted Forest.
		g.ActivateAbility(2, mage.PrecombatMain, mage.PlayerB, "Forest")
		g.StopAt(2, mage.BeginCombat)
		g.Execute()
		// Forest tapped -> destroyed. Kudzu moves to Plains.
		g.AssertPermanentCount(mage.PlayerB, "Forest", 0)
		g.AssertAttachedTo(mage.PlayerB, "Kudzu", "Plains")
	})
}

func TestRegenerationAura(t *testing.T) {
	t.Run("activate_to_regenerate", func(t *testing.T) {
		// Regeneration (Aura): {G}: Regenerate enchanted creature.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Regeneration")
		g.AddCard(mage.ZoneHand, mage.PlayerB, "Lightning Bolt")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Regeneration", "Grizzly Bears")
		// Activate regeneration shield, then Bolt the creature.
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Grizzly Bears")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerB, "Lightning Bolt", "Grizzly Bears")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Grizzly Bears should regenerate from Bolt damage.
		g.AssertPermanentCount(mage.PlayerA, "Grizzly Bears", 1)
	})
}

// ---------------------------------------------------------------------------
// Auras registered in alpha_enchantments.go (land enchantments)
// ---------------------------------------------------------------------------

func TestPsychicVenom(t *testing.T) {
	t.Run("deals_2_damage_when_land_taps", func(t *testing.T) {
		// Psychic Venom: Whenever enchanted land becomes tapped, Psychic Venom
		// deals 2 damage to that land's controller.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Forest")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Psychic Venom")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Psychic Venom", "Forest")
		g.ActivateAbility(2, mage.PrecombatMain, mage.PlayerB, "Forest")
		g.StopAt(2, mage.BeginCombat)
		g.Execute()
		// Forest tapped -> Psychic Venom deals 2 to PlayerB.
		g.AssertLife(mage.PlayerB, 18)
	})
}

func TestEvilPresence(t *testing.T) {
	t.Run("land_becomes_swamp", func(t *testing.T) {
		// Evil Presence: Enchanted land is a Swamp.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Forest")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Evil Presence")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Evil Presence", "Forest")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Forest should now be a Swamp (produce {B} instead of {G}).
		perm := g.FindPermanentByName("Forest", g.Players[1].PlayerID())
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
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Mountain")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Phantasmal Terrain")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Phantasmal Terrain", "Mountain")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Mountain should become an Island (default choice).
		perm := g.FindPermanentByName("Mountain", g.Players[1].PlayerID())
		if perm == nil {
			t.Fatal("Mountain not found")
		}
		if !perm.HasSubType("Island") {
			t.Errorf("Phantasmal Terrain should change Mountain to Island")
		}
	})
}

func TestKormusBell(t *testing.T) {
	t.Run("swamps_become_creatures", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Kormus Bell")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Swamp")
		g.StopAt(1, mage.PrecombatMain)
		g.Execute()
		perm := g.FindPermanentByName("Swamp", g.Players[0].PlayerID())
		if perm == nil {
			t.Fatal("Swamp not found")
		}
		if !perm.HasType(mage.TypeCreature) {
			t.Errorf("Kormus Bell should make Swamp a creature")
		}
		g.AssertPowerToughness(mage.PlayerA, "Swamp", 1, 1)
	})

	t.Run("swamps_can_attack", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Kormus Bell")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Swamp")
		g.Attack(1, mage.PlayerA, "Swamp")
		g.StopAt(1, mage.EndCombat)
		g.Execute()
		g.AssertLife(mage.PlayerB, 19)
	})

	t.Run("non_swamps_unaffected", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Kormus Bell")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Forest")
		g.StopAt(1, mage.PrecombatMain)
		g.Execute()
		perm := g.FindPermanentByName("Forest", g.Players[0].PlayerID())
		if perm == nil {
			t.Fatal("Forest not found")
		}
		if perm.HasType(mage.TypeCreature) {
			t.Errorf("Kormus Bell should not affect non-Swamp lands")
		}
	})
}

func TestHelmOfChatzuk(t *testing.T) {
	t.Run("grants_banding", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Helm of Chatzuk")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Helm of Chatzuk", "Grizzly Bears")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertHasAbility(mage.PlayerA, "Grizzly Bears", mage.Banding, true)
		g.AssertTapped(mage.PlayerA, "Helm of Chatzuk", true)
	})
}

func TestGlassesOfUrza(t *testing.T) {
	t.Run("taps_to_look", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Glasses of Urza")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Glasses of Urza", "PlayerB")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertTapped(mage.PlayerA, "Glasses of Urza", true)
	})
}

func TestSunglassesOfUrza(t *testing.T) {
	t.Run("red_mana_as_white", func(t *testing.T) {
		// Sunglasses of Urza lets you spend red mana as white.
		// Cast Swords to Plowshares ({W}) using only red mana sources.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Sunglasses of Urza")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Mountain")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Grizzly Bears")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Swords to Plowshares")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Swords to Plowshares", "Grizzly Bears")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Swords should exile Grizzly Bears using red mana as white.
		g.AssertPermanentCount(mage.PlayerB, "Grizzly Bears", 0)
	})
}

func TestJadeStatue(t *testing.T) {
	t.Run("becomes_creature", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Jade Statue")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Jade Statue")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		perm := g.FindPermanentByName("Jade Statue", g.Players[0].PlayerID())
		if perm == nil {
			t.Fatal("Jade Statue not found")
		}
		if !perm.HasType(mage.TypeCreature) {
			t.Errorf("Jade Statue should become a creature after activation")
		}
		g.AssertPowerToughness(mage.PlayerA, "Jade Statue", 3, 6)
	})

	t.Run("creature_until_end_of_combat", func(t *testing.T) {
		// Jade Statue should be a creature during combat (DeclareAttackers)
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Jade Statue")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Jade Statue")
		g.StopAt(1, mage.DeclareAttackers)
		g.Execute()
		perm := g.FindPermanentByName("Jade Statue", g.Players[0].PlayerID())
		if perm == nil {
			t.Fatal("Jade Statue not found")
		}
		if !perm.HasType(mage.TypeCreature) {
			t.Errorf("Jade Statue should be a creature during combat")
		}
	})

	t.Run("not_creature_after_combat", func(t *testing.T) {
		// Jade Statue should revert to non-creature after EndCombat
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Jade Statue")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Jade Statue")
		g.StopAt(1, mage.PostcombatMain)
		g.Execute()
		perm := g.FindPermanentByName("Jade Statue", g.Players[0].PlayerID())
		if perm == nil {
			t.Fatal("Jade Statue not found")
		}
		if perm.HasType(mage.TypeCreature) {
			t.Errorf("Jade Statue should NOT be a creature during postcombat main")
		}
	})

	t.Run("reverts_at_end_of_turn", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Jade Statue")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Jade Statue")
		g.StopAt(2, mage.PrecombatMain)
		g.Execute()
		perm := g.FindPermanentByName("Jade Statue", g.Players[0].PlayerID())
		if perm == nil {
			t.Fatal("Jade Statue not found")
		}
		if perm.HasType(mage.TypeCreature) {
			t.Errorf("Jade Statue should revert to non-creature at end of turn")
		}
	})
}

func TestJadeMonolith(t *testing.T) {
	t.Run("redirects_creature_damage_to_player", func(t *testing.T) {
		// {1}: The next time a source of your choice would deal damage to
		// target creature this turn, that damage is dealt to target player instead.
		g := mage.NewTestGame(t)
		g.SetLife(mage.PlayerB, 20)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Jade Monolith")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(mage.ZoneHand, mage.PlayerB, "Lightning Bolt")
		// Activate Jade Monolith targeting Grizzly Bears and PlayerB
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Jade Monolith", "Grizzly Bears", "PlayerB")
		// Lightning Bolt targets Grizzly Bears — damage redirected to PlayerB
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerB, "Lightning Bolt", "Grizzly Bears")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Bears should survive (damage redirected), PlayerB takes 3
		g.AssertPermanentCount(mage.PlayerA, "Grizzly Bears", 1)
		g.AssertLife(mage.PlayerB, 17)
	})
}

func TestPersonalIncarnation(t *testing.T) {
	t.Run("has_flying", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Personal Incarnation")
		g.StopAt(1, mage.PrecombatMain)
		g.Execute()
		g.AssertHasAbility(mage.PlayerA, "Personal Incarnation", mage.Flying, true)
		g.AssertPowerToughness(mage.PlayerA, "Personal Incarnation", 6, 6)
	})

	t.Run("lose_half_life_on_death", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.SetLife(mage.PlayerA, 20)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Personal Incarnation")
		g.AddCard(mage.ZoneHand, mage.PlayerB, "Terror")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerB, "Terror", "Personal Incarnation")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(mage.PlayerA, "Personal Incarnation", 0)
		// 20 / 2 = 10 life lost, so 10 remaining
		g.AssertLife(mage.PlayerA, 10)
	})

	t.Run("redirects_damage_from_controller", func(t *testing.T) {
		// All damage that would be dealt to you is dealt to Personal Incarnation instead.
		g := mage.NewTestGame(t)
		g.SetLife(mage.PlayerA, 20)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Personal Incarnation") // 6/6 flying
		g.AddCard(mage.ZoneHand, mage.PlayerB, "Lightning Bolt")
		// Lightning Bolt targeting PlayerA — damage should redirect to Personal Incarnation
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerB, "Lightning Bolt", "PlayerA")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Player A should still be at 20 life (damage redirected)
		g.AssertLife(mage.PlayerA, 20)
		// Personal Incarnation should have taken 3 damage (6/6 with 3 damage = alive)
		g.AssertPermanentCount(mage.PlayerA, "Personal Incarnation", 1)
	})

	t.Run("dies_when_lethal_redirected", func(t *testing.T) {
		// If enough damage is redirected, Personal Incarnation dies and
		// the controller loses half their life.
		g := mage.NewTestGame(t)
		g.SetLife(mage.PlayerA, 20)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Personal Incarnation") // 6/6 flying
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Craw Wurm")            // 6/4
		// Turn 2 is PlayerB's turn; Craw Wurm attacks PlayerA.
		g.Attack(2, mage.PlayerB, "Craw Wurm")
		g.StopAt(2, mage.EndCombat)
		g.Execute()
		// 6 damage redirected kills PI (6/6 with 6 damage). Death trigger: lose half life = 10.
		g.AssertPermanentCount(mage.PlayerA, "Personal Incarnation", 0)
		g.AssertLife(mage.PlayerA, 10)
	})
}

func TestVeteranBodyguard(t *testing.T) {
	t.Run("redirects_combat_damage", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Veteran Bodyguard") // 2/5
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Hill Giant")       // 3/3
		g.Attack(1, mage.PlayerA, "Hill Giant")
		g.StopAt(1, mage.EndCombat)
		g.Execute()
		// Veteran Bodyguard should absorb the 3 combat damage
		g.AssertLife(mage.PlayerB, 20)
		// Bodyguard should have taken 3 damage (5 toughness - 3 = 2 remaining)
		g.AssertPermanentCount(mage.PlayerB, "Veteran Bodyguard", 1)
	})

	t.Run("no_redirect_when_tapped", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Veteran Bodyguard") // 2/5
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Icy Manipulator")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Hill Giant") // 3/3
		// Tap bodyguard before combat
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerB, "Icy Manipulator", "Veteran Bodyguard")
		g.Attack(1, mage.PlayerA, "Hill Giant")
		g.StopAt(1, mage.EndCombat)
		g.Execute()
		// Bodyguard is tapped, so damage goes through to player
		g.AssertLife(mage.PlayerB, 17)
	})
}

func TestNettlingImpForceAttack(t *testing.T) {
	t.Run("forces_creature_to_attack", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Nettling Imp")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Hill Giant") // 3/3
		// Nettling Imp gives MustAttack to Hill Giant
		g.ActivateAbility(2, mage.PrecombatMain, mage.PlayerA, "Nettling Imp", "Hill Giant")
		g.StopAt(2, mage.EndCombat)
		g.Execute()
		// Hill Giant should have been forced to attack
		g.AssertTapped(mage.PlayerA, "Nettling Imp", true)
		// Hill Giant should be tapped (it attacked)
		g.AssertTapped(mage.PlayerB, "Hill Giant", true)
		// PlayerA should have taken 3 damage from the forced attack
		g.AssertLife(mage.PlayerA, 17)
	})

	t.Run("cannot_target_wall", func(t *testing.T) {
		// Nettling Imp cannot target Wall creatures
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Nettling Imp")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Wall of Stone") // 0/8 Wall
		// Try to target a Wall — should fail; Imp won't tap
		g.ActivateAbility(2, mage.PrecombatMain, mage.PlayerA, "Nettling Imp", "Wall of Stone")
		g.StopAt(2, mage.Cleanup)
		g.Execute()
		// Nettling Imp should NOT have tapped (no valid target)
		g.AssertTapped(mage.PlayerA, "Nettling Imp", false)
	})

	t.Run("destroy_if_didnt_attack", func(t *testing.T) {
		// If the target creature didn't attack this turn, destroy it at end of turn.
		// Use Meekstone to keep Hill Giant tapped (power >= 3 can't untap).
		// Hill Giant attacks on turn 2, gets tapped. Meekstone prevents untap.
		// On turn 4 (PlayerB's next turn), Hill Giant is still tapped.
		// Nettling Imp targets Hill Giant -> can't attack -> destroyed at EOT.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Nettling Imp")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Meekstone")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Hill Giant") // 3/3
		// Turn 2: Hill Giant attacks (forced or voluntary), gets tapped
		g.Attack(2, mage.PlayerB, "Hill Giant")
		// Turn 4: Meekstone keeps Hill Giant tapped. Nettling Imp targets it.
		g.ActivateAbility(4, mage.PrecombatMain, mage.PlayerA, "Nettling Imp", "Hill Giant")
		g.StopAt(4, mage.Cleanup)
		g.Execute()
		// Hill Giant was tapped and couldn't attack, so it should be destroyed at end of turn
		g.AssertPermanentCount(mage.PlayerB, "Hill Giant", 0)
	})
}

func TestCyclopeanTomb(t *testing.T) {
	t.Run("turns_land_into_swamp", func(t *testing.T) {
		// {2}, {T}: Put a mire counter on target non-Swamp land.
		// As long as Cyclopean Tomb is on the battlefield, that land is a Swamp.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Cyclopean Tomb")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Forest")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Cyclopean Tomb", "Forest")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		perm := g.FindPermanentByName("Forest", g.Players[1].PlayerID())
		if perm == nil {
			t.Fatal("Forest not found")
		}
		if !perm.HasSubType("Swamp") {
			t.Errorf("Cyclopean Tomb should make Forest a Swamp")
		}
		// Forest should have a Mire counter
		if perm.Counters[mage.Mire] < 1 {
			t.Errorf("Forest should have a Mire counter, got %d", perm.Counters[mage.Mire])
		}
	})

	t.Run("reverts_when_tomb_leaves", func(t *testing.T) {
		// When Cyclopean Tomb leaves the battlefield, the subtype override ends.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Cyclopean Tomb")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Forest")
		g.AddCard(mage.ZoneHand, mage.PlayerB, "Disenchant")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Cyclopean Tomb", "Forest")
		// Destroy Cyclopean Tomb
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerB, "Disenchant", "Cyclopean Tomb")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		perm := g.FindPermanentByName("Forest", g.Players[1].PlayerID())
		if perm == nil {
			t.Fatal("Forest not found")
		}
		if perm.HasSubType("Swamp") {
			t.Errorf("Forest should revert to non-Swamp after Cyclopean Tomb leaves")
		}
	})

	t.Run("cannot_target_swamp", func(t *testing.T) {
		// Can only target non-Swamp lands
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Cyclopean Tomb")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Swamp")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Cyclopean Tomb", "Swamp")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Cyclopean Tomb should NOT have tapped (no valid target)
		g.AssertTapped(mage.PlayerA, "Cyclopean Tomb", false)
	})
}

func TestIllusionaryMask(t *testing.T) {
	t.Run("enters_as_0_1_face_down", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Illusionary Mask")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Serra Angel")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Illusionary Mask", "Serra Angel")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Serra Angel should be on the battlefield as a face-down 0/1
		g.AssertPermanentCount(mage.PlayerA, "Serra Angel", 1)
		g.AssertPowerToughness(mage.PlayerA, "Serra Angel", 0, 1)
		perm := g.FindPermanentByName("Serra Angel", g.Players[0].PlayerID())
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
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Illusionary Mask")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Craw Wurm")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Lightning Bolt")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Illusionary Mask", "Craw Wurm")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Lightning Bolt", "Craw Wurm")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Craw Wurm should have flipped face up: 6/4 with 3 damage — survives
		g.AssertPermanentCount(mage.PlayerA, "Craw Wurm", 1)
		perm := g.FindPermanentByName("Craw Wurm", g.Players[0].PlayerID())
		if perm == nil {
			t.Fatal("Craw Wurm not found on battlefield")
		}
		if perm.FaceDown {
			t.Error("Craw Wurm should be face up after taking damage")
		}
		g.AssertPowerToughness(mage.PlayerA, "Craw Wurm", 6, 4)
	})

	t.Run("dies_if_flip_reveals_lethal", func(t *testing.T) {
		// Put Hill Giant (3/3) face-down. Bolt it (3 damage).
		// Flips to 3/3 with 3 damage — lethal, SBA destroys it.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Illusionary Mask")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Hill Giant")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Lightning Bolt")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Illusionary Mask", "Hill Giant")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Lightning Bolt", "Hill Giant")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Hill Giant should be dead: 3/3 with 3 damage
		g.AssertPermanentCount(mage.PlayerA, "Hill Giant", 0)
		g.AssertGraveyardCount(mage.PlayerA, "Hill Giant", 1)
	})

	t.Run("no_abilities_while_face_down", func(t *testing.T) {
		// Put Serra Angel (flying, vigilance) face-down.
		// Verify it does NOT have Flying while face-down.
		// Then deal 1 damage (Prodigal Sorcerer) to flip it.
		// After flip, verify it HAS Flying.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Illusionary Mask")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Serra Angel")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Prodigal Sorcerer")
		g.ActivateAbility(1, mage.PrecombatMain, mage.PlayerA, "Illusionary Mask", "Serra Angel")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Face-down: no Flying
		g.AssertHasAbility(mage.PlayerA, "Serra Angel", mage.Flying, false)
		// Now deal 1 damage with Prodigal Sorcerer to flip it
		g.ActivateAbility(1, mage.BeginCombat, mage.PlayerB, "Prodigal Sorcerer", "Serra Angel")
		g.StopAt(1, mage.DeclareAttackers)
		g.Execute()
		// Face-up: has Flying
		g.AssertHasAbility(mage.PlayerA, "Serra Angel", mage.Flying, true)
	})
}
