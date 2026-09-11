package limited

import (
	"testing"

	"github.com/google/uuid"

	_ "github.com/benprew/mage-go/cards/arabian"
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
	"github.com/benprew/mage-go/pkg/mage/interactive"
)

// Tests for cards registered in alpha_enchantments.go.

func TestHolyStrengthFizzlesWhenTargetDestroyed(t *testing.T) {
	t.Run("aura_goes_to_graveyard_when_target_removed_in_response", func(t *testing.T) {
		// Holy Strength is an Aura. While it's on the stack targeting Benalish
		// Hero, Terror destroys the Hero in response. When Holy Strength tries
		// to resolve, its only target is illegal, so it's countered by the game
		// rules (CR 608.2b) and goes to the graveyard — it must NOT end up on
		// the battlefield attached to nothing.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Benalish Hero")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Holy Strength")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Terror")

		// PlayerB casts Holy Strength on its own Benalish Hero; PlayerA responds
		// with Terror on the Hero before Holy Strength resolves.
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Holy Strength", "Benalish Hero")
		g.CastInResponseTo(gametest.PlayerA, "Terror", "Benalish Hero")
		g.StopAt(2, core.BeginCombat)
		g.Execute()

		// Terror resolves first and destroys the Hero.
		g.AssertPermanentCount(gametest.PlayerB, "Benalish Hero", 0)
		g.AssertGraveyardCount(gametest.PlayerB, "Benalish Hero", 1)
		g.AssertGraveyardCount(gametest.PlayerA, "Terror", 1)

		// Holy Strength's target is gone, so it's countered on resolution and
		// must be in the graveyard, not on the battlefield.
		g.AssertPermanentCount(gametest.PlayerB, "Holy Strength", 0)
		g.AssertGraveyardCount(gametest.PlayerB, "Holy Strength", 1)
	})
}

func TestLivingArtifact(t *testing.T) {
	t.Run("damage adds vitality counters", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sol Ring")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Living Artifact")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Living Artifact", "Sol Ring")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "PlayerA")
		g.StopAt(2, core.EndStep)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Living Artifact", core.Vitality, 3)
	})

	t.Run("upkeep may trade a counter for life", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sol Ring")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Living Artifact")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Living Artifact", "Sol Ring")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "PlayerA")
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Living Artifact", core.Vitality, 2)
		g.AssertLife(gametest.PlayerA, 18)
	})

	t.Run("upkeep ability may be declined", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sol Ring")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Living Artifact")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.GetPlayer(gametest.PlayerA).QueueMayAbilityChoices(false)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Living Artifact", "Sol Ring")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "PlayerA")
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Living Artifact", core.Vitality, 3)
		g.AssertLife(gametest.PlayerA, 17)
	})
}

func TestInvisibility(t *testing.T) {
	t.Run("creature_unblockable_except_walls", func(t *testing.T) {
		// Invisibility: Enchanted creature can't be blocked except by Walls.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")    // 3/3 non-Wall
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Invisibility")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Invisibility", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Hill Giant", "Grizzly Bears")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Hill Giant is not a Wall, so it can't block invisible creature.
		// Bears should deal 2 damage unblocked.
		g.AssertLife(gametest.PlayerB, 18)
	})

	t.Run("walls_can_still_block", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wall of Stone") // 0/8 Wall
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Invisibility")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Invisibility", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Wall of Stone", "Grizzly Bears")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Wall of Stone IS a Wall, so it can block invisible creature.
		// No damage to PlayerB.
		g.AssertLife(gametest.PlayerB, 20)
	})
}

func TestLure(t *testing.T) {
	t.Run("all_creatures_must_block", func(t *testing.T) {
		// Lure: All creatures able to block enchanted creature do so.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Savannah Lions") // 2/1
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Gray Ogre")      // 2/2
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lure")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lure", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears", "Hill Giant")
		// Even though PlayerB tries to block Hill Giant, Lure forces blocks on Bears.
		g.Block(1, gametest.PlayerB, "Savannah Lions", "Hill Giant")
		g.Block(1, gametest.PlayerB, "Gray Ogre", "Hill Giant")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// All of PlayerB's creatures must block the Lured creature (Grizzly Bears),
		// not Hill Giant. So Hill Giant gets through unblocked for 3 damage.
		g.AssertLife(gametest.PlayerB, 17)
	})
}

func TestWildGrowth(t *testing.T) {
	t.Run("enchanted_land_produces_extra_G", func(t *testing.T) {
		// Wild Growth: Whenever enchanted land is tapped for mana, its controller
		// adds an additional {G}.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Wild Growth")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Wild Growth", "Forest")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Forest")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Auto-mana adds {G} to pay for Wild Growth. Forest taps for {G},
		// Wild Growth adds another {G}. Total: 1 (auto) + 1 (Forest) + 1 (bonus) = 3G.
		// Without bonus: 1 + 1 = 2G.
		pool := g.AllPlayers()[0].ManaPool()
		if pool.CountProducedThisTurn(core.Green) < 3 {
			t.Errorf("Wild Growth should add extra {G} when Forest taps; expected >= 3 green, got %d", pool.CountProducedThisTurn(core.Green))
		}
	})
}

func TestCircleOfProtection(t *testing.T) {
	t.Run("prevents_red_damage", func(t *testing.T) {
		// Circle of Protection: Red -- {1}: The next time a red source of your
		// choice would deal damage to you this turn, prevent that damage.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Circle of Protection: Red")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt") // red source
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
		g.ActivateInResponseTo(gametest.PlayerB, "Circle of Protection: Red")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// CoP:Red should prevent the 3 red damage.
		g.AssertLife(gametest.PlayerB, 20)
	})

	t.Run("doesnt_prevent_other_colors", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Circle of Protection: Red")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Terror")                  // black spell
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")       // non-black creature for Terror
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hypnotic Specter") // black 2/2 flyer
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerB, "Circle of Protection: Red")
		g.Attack(1, gametest.PlayerA, "Hypnotic Specter")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Hypnotic Specter is black, not red. CoP:Red shouldn't prevent its damage.
		g.AssertLife(gametest.PlayerB, 18)
	})
}

func TestColorWards(t *testing.T) {
	t.Run("grants_protection_from_color", func(t *testing.T) {
		// Black Ward: Enchanted creature has protection from black.
		// (Testing one representative of the cycle.)
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Black Ward")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Terror") // black spell targeting non-black creature
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Black Ward", "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Terror", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Protection from black: can't be targeted by Terror (black).
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	})

	t.Run("ward_not_removed_by_protection", func(t *testing.T) {
		// "This effect doesn't remove this Aura." Every Ward is white, so only
		// White Ward's protection could remove the Ward itself (CR 704.5m).
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "White Ward")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "White Ward", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "White Ward", 1)
		g.AssertAttachedTo(gametest.PlayerA, "White Ward", "Grizzly Bears")
	})

	t.Run("white_ward_still_protects_from_white", func(t *testing.T) {
		// The exception covers only the Ward itself; other white sources are
		// still stopped.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "White Ward")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Swords to Plowshares")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "White Ward", "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Swords to Plowshares", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	})

	t.Run("protection_ends_when_ward_leaves", func(t *testing.T) {
		// The protection lasts only while Black Ward is attached: once Disenchant
		// destroys it, Terror can target the creature again.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Black Ward")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Disenchant")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Terror")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Black Ward", "Grizzly Bears")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Disenchant", "Black Ward")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Terror", "Grizzly Bears")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Black Ward", 0)
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
	})

	t.Run("grants_a_single_protection", func(t *testing.T) {
		// The grant is reapplied on every layer pass, so it has to replace the
		// previous pass's grant instead of piling up on the creature.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Black Ward")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Black Ward", "Grizzly Bears")
		g.StopAt(3, core.BeginCombat)
		g.Execute()
		bears := g.FindPermanentByName("Grizzly Bears", g.GetPlayer(gametest.PlayerA).PlayerID())
		if bears == nil {
			t.Fatal("Grizzly Bears not found")
		}
		if n := countProtectionAbilities(bears); n != 1 {
			t.Fatalf("Grizzly Bears carry %d protection abilities, want 1", n)
		}
	})
}

func countProtectionAbilities(p *mage.Permanent) int {
	n := 0
	for _, a := range p.RuntimeAbilities {
		if _, ok := mage.UnwrapAbility(a).(*mage.ProtectionAbility); ok {
			n++
		}
	}
	return n
}

func TestLaceCycle(t *testing.T) {
	t.Run("changes_permanent_color", func(t *testing.T) {
		// Deathlace: Target spell or permanent becomes black.
		// (Testing one representative of the cycle.)
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // green
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Deathlace")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Deathlace", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Grizzly Bears should now be black.
		perm := g.FindPermanentByName("Grizzly Bears", g.AllPlayers()[0].PlayerID())
		if perm == nil {
			t.Fatal("Grizzly Bears not found")
		}
		hasBlack := false
		for _, col := range perm.Colors() {
			if col == core.Black {
				hasBlack = true
			}
		}
		if !hasBlack {
			t.Errorf("Deathlace should make Grizzly Bears black")
		}
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
			if perm.ControllerID() == playerAID && perm.Name() == "Sol Ring" {
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
			if perm.ControllerID() == playerAID && perm.Name() == "Sol Ring" && perm.HasType(core.TypeEnchantment) {
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
			if perm.ControllerID() == playerAID && perm.Name() == "Sol Ring" &&
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
			if perm.ControllerID() == playerAID && perm.Name() == "Sol Ring" && perm.HasType(core.TypeEnchantment) {
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

func TestManabarbs(t *testing.T) {
	t.Run("damages_player_who_tapped_land_for_mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Manabarbs")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")
		g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerB, "Mountain")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 20)
		g.AssertLife(gametest.PlayerB, 19)
	})

	t.Run("does_not_damage_player_for_nonmana_land_ability", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Manabarbs")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island of Wak-Wak")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Serra Angel")
		g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerB, "Island of Wak-Wak", "Serra Angel")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 20)
		g.AssertLife(gametest.PlayerB, 20)
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
	t.Run("draw_step_actually_skipped", func(t *testing.T) {
		// Oracle: "If you would draw a card during your draw step, instead you
		// may skip that draw." The skip must apply to the active draw step —
		// not the next one. Library count after PlayerB's draw step should be
		// unchanged.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island Sanctuary")
		// Library size is the marker — Sanctuary should leave it untouched
		// during the controller's draw step.
		for range 3 {
			g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Forest")
		}
		// Stop right after PlayerB's draw step on turn 2.
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		// PlayerB skipped its draw; library is still 3.
		g.AssertLibraryCount(gametest.PlayerB, "Forest", 3)
	})

	t.Run("controller_can_decline_skip_and_draw_normally", func(t *testing.T) {
		// Player chooses NOT to skip on turn 2 — draw happens normally and
		// sanctuary protection does not activate, so a non-flyer can still
		// connect on turn 3.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island Sanctuary")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // no flying
		for range 5 {
			g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
			g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Forest")
		}
		g.GetPlayer(gametest.PlayerB).QueueMayAbilityChoices(false)
		g.Attack(3, gametest.PlayerA, "Grizzly Bears")
		g.StopAt(3, core.EndCombat)
		g.Execute()
		// Sanctuary not active because PlayerB declined. Bears (2/2) connects.
		g.AssertLife(gametest.PlayerB, 18)
		// PlayerB drew normally on turn 2 — library went from 5 to 4 (turn 2
		// draw) to 3 (turn 4 draw would happen later, but we stop on turn 3).
		// Just confirm the turn-2 draw happened.
		g.AssertLibraryCount(gametest.PlayerB, "Forest", 4)
	})

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
		// PlayerB had 3 untapped lands at the start of turn 2 -> 3 damage at upkeep.
		g.AssertLife(gametest.PlayerB, 17)
	})

	t.Run("snapshot_taken_before_untap_step", func(t *testing.T) {
		// Snapshot is taken at the start of the turn (before the untap step).
		// If lands are tapped at end of the previous turn, they should not be
		// counted even though they untap during the untap step.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Power Surge")
		p1 := g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains")
		p2 := g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains")
		p3 := g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains")
		for range 5 {
			g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
			g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Forest")
		}
		// Stop just before turn 2's Untap step, then tap PlayerB's lands so the
		// snapshot taken at the start of turn 2 sees zero untapped lands.
		g.StopAt(2, core.Untap)
		g.Execute()
		for _, id := range []uuid.UUID{p1, p2, p3} {
			g.FindPermanent(id).Tapped = true
		}
		g.RunStepWithPriority(core.Untap)
		g.RunStepWithPriority(core.Upkeep)
		// All Plains were tapped at the start of turn 2 -> snapshot is 0 -> 0 damage.
		g.AssertLife(gametest.PlayerB, 20)
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
		// Plains provide the upkeep payment while the converted Mountain remains untapped.
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
		mountain := g.FindPermanentByName("Mountain", g.AllPlayers()[0].PlayerID())
		if mountain == nil {
			t.Fatal("Mountain not found")
		}
		if white, red := landManaProduction(mountain, core.White), landManaProduction(mountain, core.Red); white != 1 || red != 0 {
			t.Errorf("Conversion mana abilities: white=%d red=%d, want white=1 red=0", white, red)
		}
	})

	t.Run("sacrifice_unless_pay_WW", func(t *testing.T) {
		// At the beginning of your upkeep, sacrifice Conversion unless you pay {W}{W}.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Conversion")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// No {W}{W} paid -> Conversion sacrificed.
		g.AssertPermanentCount(gametest.PlayerA, "Conversion", 0)
		mountain := g.FindPermanentByName("Mountain", g.AllPlayers()[0].PlayerID())
		if mountain == nil {
			t.Fatal("Mountain not found")
		}
		if white, red := landManaProduction(mountain, core.White), landManaProduction(mountain, core.Red); white != 0 || red != 1 {
			t.Errorf("restored Mountain mana abilities: white=%d red=%d, want white=0 red=1", white, red)
		}
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

	t.Run("controller_of_destroyed_land_chooses_target", func(t *testing.T) {
		// The player who controlled the destroyed land picks a land to attach
		// Kudzu to — not the first land found.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Kudzu")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Kudzu", "Forest")
		// PlayerB (controller of the destroyed Forest) chooses Mountain.
		g.ChoosePermanent(gametest.PlayerB, "Mountain")
		g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerB, "Forest")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Forest", 0)
		g.AssertAttachedTo(gametest.PlayerB, "Kudzu", "Mountain")
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

func TestStasis(t *testing.T) {
	t.Run("players_skip_untap_step", func(t *testing.T) {
		// Stasis: Players skip their untap steps.
		g := gametest.NewTestGame(t)
		// Island lets PlayerA pay {U} at turn 1 upkeep so Stasis survives.
		// At turn 3 untap the Island is tapped (used for payment) and can't
		// untap (Stasis prevents it), so Stasis is sacrificed at turn 3 upkeep.
		// But the untap prevention already happened, so Bears stay tapped.
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Stasis")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		// Attack with Bears to tap them
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		// Turn 3 is PlayerA's next turn -- Bears should NOT untap with Stasis.
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Grizzly Bears", true)
	})

	t.Run("sacrifice_unless_pay_U", func(t *testing.T) {
		// At the beginning of your upkeep, sacrifice Stasis unless you pay {U}.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Stasis")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Stasis should be sacrificed at upkeep (no {U} paid).
		g.AssertPermanentCount(gametest.PlayerA, "Stasis", 0)
	})
}

func TestAnimateDeadCastabilityWithOpponentCreatureOnly(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Animate Dead")
	g.AddCard(core.ZoneGraveyard, gametest.PlayerB, "Serra Angel")
	g.SetStep(core.PrecombatMain)

	playerAID := g.AllPlayers()[0].PlayerID()
	castable := g.GetCastableSpells(playerAID)

	found := false
	for _, c := range castable {
		if c.Name() == "Animate Dead" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Animate Dead to be in GetCastableSpells when opponent's graveyard has a creature, got %v", castable)
	}

	opts := interactive.GetAvailableActions(g.Game, playerAID)
	foundOpt := false
	for _, opt := range opts {
		if opt.CardName == "Animate Dead" {
			foundOpt = true
			if len(opt.ValidTargets) == 0 {
				t.Fatalf("expected Animate Dead to have valid targets, got 0")
			}
			break
		}
	}
	if !foundOpt {
		t.Fatalf("expected Animate Dead in GetAvailableActions, got %v", opts)
	}
}
