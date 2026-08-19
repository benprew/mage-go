package limited

import (
	"testing"

	"github.com/google/uuid"

	_ "github.com/benprew/mage-go/cards/arabian"
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

// Tests for cards registered in alpha_artifacts.go (and a few
// from alpha_spells.go / alpha_enchantments.go that are closely related).

// ---------------------------------------------------------------------------
// Artifacts

func landManaProduction(perm *mage.Permanent, color core.Color) int {
	total := 0
	for _, ability := range perm.RuntimeAbilities {
		for _, production := range mage.ManaProductionsForAbility(ability) {
			if production.Color == color {
				total += production.Amount
			}
		}
	}
	return total
}

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
			if perm.ControllerID() == playerAID && perm.HasType(core.TypeLand) && perm.Tapped {
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
			if perm.ControllerID() == playerBID && perm.HasType(core.TypeLand) && perm.Tapped {
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

func TestDisruptingScepterOnlyDuringControllersTurn(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Disrupting Scepter")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 3)
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
	g.StopAt(2, core.PrecombatMain)
	g.Execute()

	playerA := g.GetPlayer(gametest.PlayerA).PlayerID()
	playerB := g.GetPlayer(gametest.PlayerB).PlayerID()
	scepter := g.FindPermanentByName("Disrupting Scepter", playerA)
	if scepter == nil {
		t.Fatal("Disrupting Scepter not found")
	}
	if err := g.ActivateAbilityByIndex(playerA, scepter.ID(), 0, []uuid.UUID{playerB}); err == nil {
		t.Fatal("Disrupting Scepter should not be activatable during an opponent's turn")
	}
	g.AssertTapped(gametest.PlayerA, "Disrupting Scepter", false)
}

func TestAnkhOfMishra(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ankh of Mishra")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Forest")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Island")
	g.StopAt(2, core.BeginCombat)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 18)
	g.AssertLife(gametest.PlayerB, 18)
}

func TestDingusEgg(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dingus Egg")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Sinkhole")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Sinkhole", "Forest")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 18)
	g.AssertLife(gametest.PlayerA, 20)
}

func TestLibraryOfLeng(t *testing.T) {
	t.Run("removes maximum hand size", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Library of Leng")
		playerA := g.GetPlayer(gametest.PlayerA).PlayerID()
		if got := g.MaximumHandSize(playerA); got != -1 {
			t.Fatalf("maximum hand size = %d, want no maximum", got)
		}
	})

	t.Run("effect discard may go on top of library", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Library of Leng")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Mind Twist")
		g.ChooseDiscard(gametest.PlayerA, "Grizzly Bears")
		g.GetPlayer(gametest.PlayerA).QueueMayAbilityChoices(true)
		g.CastSpellWithX(2, core.PrecombatMain, gametest.PlayerB, "Mind Twist", 1, "PlayerA")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 0)
		g.AssertLibraryTop(gametest.PlayerA, "Grizzly Bears")
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
		// Forcefield: {1}: The next time an unblocked creature of your choice
		// would deal combat damage to you this turn, prevent all but 1 of
		// that damage.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forcefield")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Craw Wurm") // 6/4
		g.ActivateAbility(1, core.DeclareBlockers, gametest.PlayerB, "Forcefield", "Craw Wurm")
		g.Attack(1, gametest.PlayerA, "Craw Wurm")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Forcefield should reduce 6 unblocked damage to 1.
		g.AssertLife(gametest.PlayerB, 19)
	})
}

func TestForcefieldMultipleAttackers(t *testing.T) {
	// Forcefield Oracle: "{1}: The next time an unblocked creature of your
	// choice would deal combat damage to you this turn, prevent all but 1
	// of that damage." Only ONE chosen creature has its damage reduced —
	// other unblocked attackers still deal their full damage.
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forcefield")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Craw Wurm")     // 6/4
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
	g.ActivateAbility(1, core.DeclareBlockers, gametest.PlayerB, "Forcefield", "Craw Wurm")
	g.Attack(1, gametest.PlayerA, "Craw Wurm", "Grizzly Bears")
	g.StopAt(1, core.EndCombat)
	g.Execute()
	// Craw Wurm 6 -> 1 (chosen), Grizzly Bears 2 (unchanged) = 3 damage total.
	g.AssertLife(gametest.PlayerB, 17)
}

func TestForcefieldBlockedCreature(t *testing.T) {
	t.Run("does_not_reduce_blocked_damage", func(t *testing.T) {
		// Forcefield only prevents unblocked combat damage.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forcefield")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Craw Wurm")     // 6/4
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // 2/2
		g.ActivateAbility(1, core.DeclareBlockers, gametest.PlayerB, "Forcefield", "Craw Wurm")
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
		g.ActivateAbility(1, core.DeclareBlockers, gametest.PlayerB, "Forcefield", "Llanowar Elves")
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

func TestConservator(t *testing.T) {
	t.Run("prevents_2_combat_damage_to_you_without_a_target", func(t *testing.T) {
		// Conservator: {3}, {T}: Prevent the next 2 damage that would be
		// dealt to you this turn. The ability is not targeted — it always
		// applies to "you".
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Conservator")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3
		// PlayerA activates during PlayerB's turn so the prevention applies
		// to the incoming combat damage. No target argument is passed.
		g.ActivateAbility(2, core.BeginCombat, gametest.PlayerA, "Conservator")
		g.Attack(2, gametest.PlayerB, "Hill Giant")
		g.StopAt(2, core.EndCombat)
		g.Execute()
		// 2 of the 3 damage prevented, 1 gets through.
		g.AssertLife(gametest.PlayerA, 19)
		g.AssertTapped(gametest.PlayerA, "Conservator", true)
	})
}

func TestForcedActivationTargets(t *testing.T) {
	t.Run("controller_target_is_forced", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		controller := g.AllPlayers()[0].PlayerID()
		bears := g.FindPermanentByName("Grizzly Bears", controller)
		forced := mage.ForcedActivationTargets(controller, bears.Card,
			[]mage.Target{mage.TargetController()}, g.Game)
		if len(forced) != 1 || forced[0] != controller {
			t.Fatalf("expected [controller], got %v", forced)
		}
	})

	t.Run("single_legal_creature_is_forced", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		controller := g.AllPlayers()[0].PlayerID()
		bears := g.FindPermanentByName("Grizzly Bears", controller)
		forced := mage.ForcedActivationTargets(controller, bears.Card,
			[]mage.Target{mage.TargetCreature()}, g.Game)
		if len(forced) != 1 || forced[0] != bears.ID() {
			t.Fatalf("expected [bears], got %v", forced)
		}
	})

	t.Run("multiple_legal_creatures_is_a_real_choice", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		controller := g.AllPlayers()[0].PlayerID()
		bears := g.FindPermanentByName("Grizzly Bears", controller)
		if forced := mage.ForcedActivationTargets(controller, bears.Card,
			[]mage.Target{mage.TargetCreature()}, g.Game); forced != nil {
			t.Fatalf("expected nil (prompt the player), got %v", forced)
		}
	})
}

// ---------------------------------------------------------------------------
// Instants/Sorceries registered in alpha_artifacts.go
// ---------------------------------------------------------------------------

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
		if black, green := landManaProduction(perm, core.Black), landManaProduction(perm, core.Green); black != 1 || green != 0 {
			t.Errorf("Evil Presence mana abilities: black=%d green=%d, want black=1 green=0", black, green)
		}
	})

	t.Run("land_reverts_when_aura_leaves", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Evil Presence")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Evil Presence", "Forest")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		aura := g.FindPermanentByName("Evil Presence", g.GetPlayer(gametest.PlayerA).PlayerID())
		if aura == nil {
			t.Fatal("Evil Presence not found")
		}
		g.DestroyPermanent(aura)

		land := g.FindPermanentByName("Forest", g.GetPlayer(gametest.PlayerB).PlayerID())
		if land == nil {
			t.Fatal("Forest not found")
		}
		if land.HasSubType("Swamp") || !land.HasSubType("Forest") {
			t.Fatal("Forest did not regain its printed subtype")
		}
		if black, green := landManaProduction(land, core.Black), landManaProduction(land, core.Green); black != 0 || green != 1 {
			t.Errorf("restored Forest mana abilities: black=%d green=%d, want black=0 green=1", black, green)
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
		if blue, red := landManaProduction(perm, core.Blue), landManaProduction(perm, core.Red); blue != 1 || red != 0 {
			t.Errorf("Phantasmal Terrain mana abilities: blue=%d red=%d, want blue=1 red=0", blue, red)
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
		if black, green := landManaProduction(perm, core.Black), landManaProduction(perm, core.Green); black != 1 || green != 0 {
			t.Errorf("Phantasmal Terrain mana abilities: black=%d green=%d, want black=1 green=0", black, green)
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

	t.Run("cannot attack the turn it was cast", func(t *testing.T) {
		// Per CR 302.1, a creature can't attack unless its controller has
		// continuously controlled it since their most recent turn began. A
		// Jade Statue cast and animated on the same turn is summoning-sick.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Black Lotus")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sol Ring")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Jade Statue")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Jade Statue")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Jade Statue")
		g.Attack(1, gametest.PlayerA, "Jade Statue")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Sickness prevents the attack — PlayerB takes no damage.
		g.AssertLife(gametest.PlayerB, 20)
	})

	t.Run("can attack the turn after it was cast", func(t *testing.T) {
		// On the controller's next turn, sickness has cleared at untap; the
		// animated Jade Statue can attack normally.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Black Lotus")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sol Ring")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Jade Statue")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Jade Statue")
		g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Jade Statue")
		g.Attack(3, gametest.PlayerA, "Jade Statue")
		g.StopAt(3, core.EndCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 17)
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

	t.Run("cannot_activate_on_controllers_turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Nettling Imp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Nettling Imp", "Hill Giant")
		g.StopAt(1, core.Cleanup)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Nettling Imp", false)
		g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 1)
	})

	t.Run("cannot_target_summoning_sick_creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Nettling Imp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		hillGiant := g.FindPermanentByName("Hill Giant", g.AllPlayers()[1].PlayerID())
		if hillGiant == nil {
			t.Fatal("Hill Giant not found")
		}
		hillGiant.GrantBaseAttr(core.AttrSummonSick)
		imp := g.FindPermanentByName("Nettling Imp", g.AllPlayers()[0].PlayerID())
		if imp == nil {
			t.Fatal("Nettling Imp not found")
		}
		if err := g.ActivateAbilityByIndex(g.AllPlayers()[0].PlayerID(), imp.ID(), 0, []uuid.UUID{hillGiant.ID()}); err == nil {
			t.Fatal("expected Nettling Imp activation to reject summoning-sick target")
		}
		g.AssertTapped(gametest.PlayerA, "Nettling Imp", false)
		g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 1)
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
		if black, green := landManaProduction(perm, core.Black), landManaProduction(perm, core.Green); black != 1 || green != 0 {
			t.Errorf("Cyclopean Tomb mana abilities: black=%d green=%d, want black=1 green=0", black, green)
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
		if black, green := landManaProduction(perm, core.Black), landManaProduction(perm, core.Green); black != 0 || green != 1 {
			t.Errorf("restored Forest mana abilities: black=%d green=%d, want black=0 green=1", black, green)
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
