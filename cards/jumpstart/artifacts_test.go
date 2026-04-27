package jumpstart

import (
	"testing"

	_ "git.sr.ht/~cdcarter/mage-go/cards/limited"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

func TestArcaneEncyclopedia(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Arcane Encyclopedia")
	for i := 0; i < 5; i++ {
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
		for i := 0; i < 5; i++ {
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
		for i := 0; i < 3; i++ {
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
	for i := 0; i < 3; i++ {
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
	for i := 0; i < 3; i++ {
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
		for i := 0; i < 3; i++ {
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
		for i := 0; i < 3; i++ {
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
		for i := 0; i < 3; i++ {
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
	for i := 0; i < 3; i++ {
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
	}
	g.Attack(1, gametest.PlayerA, "Hill Giant")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 17)
	g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
}

func TestWarmongersChariot(t *testing.T) {
	g := gametest.NewTestGame(t)
	chariotID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Warmonger's Chariot")
	wallID := g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Wood")
	g.Attach(chariotID, wallID)
	g.Attack(1, gametest.PlayerA, "Wall of Wood")
	g.StopAt(1, core.EndStep)
	g.Execute()
	g.AssertLife(gametest.PlayerB, 18)
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
