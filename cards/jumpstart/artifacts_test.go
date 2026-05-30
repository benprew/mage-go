package jumpstart

import (
	"testing"

	_ "github.com/benprew/mage-go/cards/limited"
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestArcaneEncyclopedia(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Arcane Encyclopedia")
	for range 5 {
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
	}
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Arcane Encyclopedia")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertTapped(gametest.PlayerA, "Arcane Encyclopedia", true)
}

func TestDreamstoneHedron(t *testing.T) {
	t.Run("sacrifice_draws_three", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dreamstone Hedron")
		for range 5 {
			g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		}
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Dreamstone Hedron")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 3)
		g.AssertGraveyardCount(gametest.PlayerA, "Dreamstone Hedron", 1)
	})
}

func TestUnstableObelisk(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Unstable Obelisk")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Serra Angel")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Unstable Obelisk", "Serra Angel")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerB, "Serra Angel", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Unstable Obelisk", 1)
}

func TestAetherSpellbomb(t *testing.T) {
	t.Run("bounces_creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Aether Spellbomb")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Aether Spellbomb", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertHandCount(gametest.PlayerB, "Grizzly Bears", 1)
		g.AssertGraveyardCount(gametest.PlayerA, "Aether Spellbomb", 1)
	})
	t.Run("draws_card", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Aether Spellbomb")
		for range 3 {
			g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		}
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Aether Spellbomb", "Aether Spellbomb")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
		g.AssertGraveyardCount(gametest.PlayerA, "Aether Spellbomb", 1)
	})
}

func TestChromaticSphere(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Chromatic Sphere")
	for range 3 {
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
	}
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Chromatic Sphere")
	g.ChooseManaColor(gametest.PlayerA, core.Blue)
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
	g.AssertGraveyardCount(gametest.PlayerA, "Chromatic Sphere", 1)
}

func TestBubblingCauldron(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Bubbling Cauldron")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.SetLife(gametest.PlayerA, 10)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Bubbling Cauldron")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerA, 14)
	g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestGuardianIdol(t *testing.T) {
	t.Run("enters_tapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Guardian Idol")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Guardian Idol")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Guardian Idol", true)
	})
	t.Run("becomes_creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Guardian Idol")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Guardian Idol")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Guardian Idol", 2, 2)
	})
}

func TestProphethicPrism(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Prophetic Prism")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	for range 3 {
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
	}
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Prophetic Prism")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestTerrarion(t *testing.T) {
	t.Run("enters_tapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Terrarion")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Terrarion")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Terrarion", true)
	})
	t.Run("activated_ability_sacs_for_two_mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Terrarion")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Terrarion")
		g.ChooseManaColor(gametest.PlayerA, core.Blue)
		g.ChooseManaColor(gametest.PlayerA, core.Blue)
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Terrarion", 1)
	})
	t.Run("destroyed_draws_card", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Terrarion")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Shatter")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain", 3)
		for range 3 {
			g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		}
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Shatter", "Terrarion")
		g.StopAt(2, core.EndStep)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Terrarion", 1)
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}

func TestScrollOfAvacyn(t *testing.T) {
	t.Run("draws_card_no_angel", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Scroll of Avacyn")
		for range 3 {
			g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		}
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Scroll of Avacyn")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
		g.AssertLife(gametest.PlayerA, 20)
	})
	t.Run("draws_and_gains_5_with_angel", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Scroll of Avacyn")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Serra Angel")
		for range 3 {
			g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		}
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Scroll of Avacyn")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
		g.AssertLife(gametest.PlayerA, 25)
	})
}

func TestMaraudersAxe(t *testing.T) {
	g := gametest.NewTestGame(t)
	axeID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Marauder's Axe")
	bearID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.Attach(axeID, bearID)
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 4, 2)
}

func TestPiratesCutlass(t *testing.T) {
	t.Run("etb_attaches_to_pirate", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Pirate's Cutlass")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 3)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kitesail Corsair")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Pirate's Cutlass")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertAttachedTo(gametest.PlayerA, "Pirate's Cutlass", "Kitesail Corsair")
	})
	t.Run("boosts_when_attached", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		cutlassID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Pirate's Cutlass")
		bearID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.Attach(cutlassID, bearID)
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 4, 3)
	})
}

func TestRoguesGloves(t *testing.T) {
	g := gametest.NewTestGame(t)
	glovesID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rogue's Gloves")
	giantID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	g.Attach(glovesID, giantID)
	for range 3 {
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
	}
	g.Attack(1, gametest.PlayerA, "Hill Giant")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 17)
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestWarmongersChariot(t *testing.T) {
	t.Run("equipped defender can attack", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		chariotID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Warmonger's Chariot")
		wallID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Wood")
		g.Attach(chariotID, wallID)
		g.Attack(1, gametest.PlayerA, "Wall of Wood")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 18)
	})

	t.Run("equipped non-defender still gets +2/+2 boost", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		chariotID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Warmonger's Chariot")
		bearID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.Attach(chariotID, bearID)
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 4, 4)
	})
}

func TestHeraldsHorn(t *testing.T) {
	t.Run("cost_reduction_only_for_chosen_type", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.ChooseString(gametest.PlayerA, "Bear")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Herald's Horn")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 5)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Herald's Horn")
		g.StopAt(1, core.EndStep)
		g.Execute()

		// Herald's Horn entered and recorded "Bear" as the chosen type.
		pid := g.GetPlayer(gametest.PlayerA).PlayerID()
		hornPerm := g.FindPermanentByName("Herald's Horn", pid)
		if hornPerm == nil {
			t.Fatalf("Herald's Horn not on battlefield")
		}
		if hornPerm.ChosenSubtype != "Bear" {
			t.Fatalf("ChosenSubtype: got %q, want %q", hornPerm.ChosenSubtype, "Bear")
		}

		// Stage hand cards to inspect ConditionalSpellCostReduction.
		bearCard, err := mage.CreateCard("Grizzly Bears")
		if err != nil {
			t.Fatalf("create Grizzly Bears: %v", err)
		}
		bearCard.SetOwner(pid)
		g.GetPlayer(gametest.PlayerA).AddToHand(bearCard)

		goblinCard, err := mage.CreateCard("Mons's Goblin Raiders")
		if err != nil {
			t.Fatalf("create Mons's Goblin Raiders: %v", err)
		}
		goblinCard.SetOwner(pid)
		g.GetPlayer(gametest.PlayerA).AddToHand(goblinCard)

		if got := g.ConditionalSpellCostReduction(pid, bearCard); got != 1 {
			t.Errorf("Bear (chosen) reduction: got %d, want 1", got)
		}
		if got := g.ConditionalSpellCostReduction(pid, goblinCard); got != 0 {
			t.Errorf("Goblin (not chosen) reduction: got %d, want 0", got)
		}
	})

	t.Run("upkeep_reveals_chosen_creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.ChooseString(gametest.PlayerA, "Bear")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Herald's Horn")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain")
		g.StopAt(2, core.PrecombatMain)
		g.Execute()

		// Upkeep on turn 2 sees Grizzly Bears on top, may-prompt defaults to
		// true, so it goes to hand. Mountain remains as the new top.
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
		g.AssertLibraryCount(gametest.PlayerA, "Grizzly Bears", 0)
		g.AssertLibraryTop(gametest.PlayerA, "Mountain")
	})

	t.Run("upkeep_does_not_reveal_non_creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.ChooseString(gametest.PlayerA, "Bear")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Herald's Horn")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.StopAt(2, core.PrecombatMain)
		g.Execute()

		// Top is Mountain (not a creature, not a Bear). Hand and library
		// are unchanged with respect to this trigger.
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 0)
		g.AssertHandCount(gametest.PlayerA, "Mountain", 0)
		g.AssertLibraryTop(gametest.PlayerA, "Mountain", "Grizzly Bears")
	})

	t.Run("upkeep_does_not_reveal_wrong_subtype", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.ChooseString(gametest.PlayerA, "Bear")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Herald's Horn")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mons's Goblin Raiders")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.StopAt(2, core.PrecombatMain)
		g.Execute()

		g.AssertHandCount(gametest.PlayerA, "Mons's Goblin Raiders", 0)
		g.AssertLibraryTop(gametest.PlayerA, "Mons's Goblin Raiders", "Grizzly Bears")
	})

	t.Run("upkeep_player_declines_reveal", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.ChooseString(gametest.PlayerA, "Bear")
		g.GetPlayer(gametest.PlayerA).QueueMayAbilityChoices(false)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Herald's Horn")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain")
		g.StopAt(2, core.PrecombatMain)
		g.Execute()

		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 0)
		g.AssertLibraryTop(gametest.PlayerA, "Grizzly Bears", "Mountain")
	})
}

func TestManaGeode(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Mana Geode")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 3)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Mountain")
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
	g.ChooseScry(gametest.PlayerA, []string{"Mountain"}, nil)
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Mana Geode")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLibraryTop(gametest.PlayerA, "Forest", "Mountain")
}

// Terrarion: when the artifact is sacrificed (via its own activated ability),
// the LtB trigger draws a card.
func TestTerrarion_SacrificeDrawsCard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Terrarion")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain", 2)
	g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears", 3)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Terrarion")
	g.ChooseManaColor(gametest.PlayerA, core.Blue)
	g.ChooseManaColor(gametest.PlayerA, core.Blue)
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertGraveyardCount(gametest.PlayerA, "Terrarion", 1)
	// Drew a Grizzly Bears (default agent doesn't auto-cast spells).
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
}
