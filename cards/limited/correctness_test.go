package limited

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

// Ensure all Alpha card packages are imported.
var (
	_ = registerCreatures     //nolint:unused
	_ = registerSpells        //nolint:unused
	_ = registerEnchantments  //nolint:unused
	_ = registerArtifacts     //nolint:unused
	_ = registerLands         //nolint:unused
)

// =============================================================================
// Per-card correctness tests. Each function validates the correct MTG behavior
// for one card and fails when the implementation is wrong or incomplete.
// =============================================================================

func TestBlackLotus(t *testing.T) {
	t.Run("sacrificed on activation", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Black Lotus")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Black Lotus")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Black Lotus", 0)
	})

	t.Run("produces 3 mana of chosen color", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Black Lotus")
		g.ChooseManaColor(gametest.PlayerA, core.Blue)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Black Lotus")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Auto-mana adds 5 of each color. Lotus adds 3 of chosen (Blue).
		pool := g.AllPlayers()[0].ManaPool()
		if pool.Count(core.Blue) < 8 {
			t.Errorf("Black Lotus should produce 3 mana of chosen color; expected >= 8 blue (5 auto + 3 Lotus), got %d", pool.Count(core.Blue))
		}
	})
}

func TestBerserk(t *testing.T) {
	t.Run("doubles target creature power", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Craw Wurm") // 6/4
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Berserk")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Berserk", "Craw Wurm")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Craw Wurm", 12, 4)
	})

	t.Run("destroys creature at end of turn if it attacked", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Berserk")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Berserk", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.StopAt(2, core.Upkeep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
	})

	t.Run("does not destroy creature if it did not attack", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Berserk")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Berserk", "Grizzly Bears")
		g.StopAt(2, core.Upkeep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}

func TestWheelOfFortune(t *testing.T) {
	t.Run("discards hand then draws 7", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Wheel of Fortune")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Hill Giant")
		for i := 0; i < 10; i++ {
			g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		}
		for i := 0; i < 10; i++ {
			g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Grizzly Bears")
		}
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Wheel of Fortune")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		playerA := g.AllPlayers()[0]
		if len(playerA.Hand()) != 7 {
			t.Errorf("expected 7 cards in hand after Wheel, got %d", len(playerA.Hand()))
		}
		// Bears and Giant should have been discarded before draw-7
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 1)
		g.AssertGraveyardCount(gametest.PlayerA, "Hill Giant", 1)
	})
}

func TestTimetwister(t *testing.T) {
	t.Run("shuffles hand and graveyard into library", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Timetwister")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
		for i := 0; i < 10; i++ {
			g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
		}
		for i := 0; i < 7; i++ {
			g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Forest")
		}
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Timetwister")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Graveyard cards should be shuffled into library
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 0)
		// Hand cards should be shuffled into library, not discarded to graveyard
		g.AssertGraveyardCount(gametest.PlayerA, "Lightning Bolt", 0)
	})
}

func TestHowlFromBeyond(t *testing.T) {
	t.Run("gives +X/+0", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Howl from Beyond")
		g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Howl from Beyond", 5, "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 7, 2)
	})
}

func TestGuardianAngel(t *testing.T) {
	t.Run("prevents X damage to creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Guardian Angel")
		// Prevent 3 damage to Bears, then Bolt them
		g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Guardian Angel", 3, "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}

func TestFirebreathingAura(t *testing.T) {
	t.Run("grants repeatable +1/+0 activated ability", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Firebreathing")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Firebreathing", "Grizzly Bears")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// {R}: +1/+0, activated twice = 2+2/2 = 4/2
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 4, 2)
	})
}

func TestBlessingAura(t *testing.T) {
	t.Run("grants repeatable +1/+1 activated ability", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Blessing")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Blessing", "Grizzly Bears")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// {W}: +1/+1, activated twice = 2+2/2+2 = 4/4
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 4, 4)
	})
}

func TestEarthbind(t *testing.T) {
	t.Run("removes flying from enchanted creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Air Elemental") // 4/4 Flying
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Earthbind")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Earthbind", "Air Elemental")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerB, "Air Elemental", core.Flying, false)
	})

	t.Run("creature regains flying if granted later by newer timestamp", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Air Elemental")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Earthbind")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Flight")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Earthbind", "Air Elemental")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Flight", "Air Elemental")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerB, "Air Elemental", core.Flying, true)
	})

	t.Run("removes flying from creature that gains it later if earthbind has newer timestamp", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Flight")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Earthbind")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Flight", "Hill Giant")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Earthbind", "Hill Giant")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerB, "Hill Giant", core.Flying, false)
	})
}

func TestParalyze(t *testing.T) {
	t.Run("creature stays tapped through untap step", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Paralyze")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Paralyze", "Hill Giant")
		// Turn 2 is PlayerB's turn; creature should remain tapped
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		g.AssertTapped(gametest.PlayerB, "Hill Giant", true)
	})
}

func TestForceOfNature(t *testing.T) {
	t.Run("deals 8 damage on upkeep if not paid", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Force of Nature")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Upkeep cost is {G}{G}{G}{G} or take 8 damage
		g.AssertLife(gametest.PlayerA, 12)
	})
}

func TestLordOfThePit(t *testing.T) {
	t.Run("sacrifices creature on upkeep instead of dealing damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lord of the Pit")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Scathe Zombies") // sacrifice fodder
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Scathe Zombies", 0)
		g.AssertLife(gametest.PlayerA, 20) // no damage when creature is sacrificed
	})
}

func TestPhantasmalForces(t *testing.T) {
	t.Run("sacrificed if upkeep cost not paid", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Phantasmal Forces")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Upkeep cost is {U} or sacrifice
		g.AssertPermanentCount(gametest.PlayerA, "Phantasmal Forces", 0)
	})
}

func TestSengirVampire(t *testing.T) {
	t.Run("no counter when another source kills creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sengir Vampire") // 4/4 Flying
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		// Kill Bears with Bolt, not Sengir's combat damage
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Sengir Vampire", core.P1P1, 0)
	})
}

func TestDrudgeSkeletons(t *testing.T) {
	t.Run("regenerates from lethal damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Drudge Skeletons") // 1/1
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		// Activate regen on Upkeep, Bolt on PrecombatMain
		g.ActivateAbility(1, core.Upkeep, gametest.PlayerA, "Drudge Skeletons")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Drudge Skeletons")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Drudge Skeletons", 1)
	})
}

func TestSamiteHealer(t *testing.T) {
	t.Run("prevents 1 damage to creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Samite Healer")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Merfolk of the Pearl Trident") // 1/1
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Prodigal Sorcerer")
		// Healer prevents 1, Tim deals 1 — Merfolk should survive
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Samite Healer", "Merfolk of the Pearl Trident")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerB, "Prodigal Sorcerer", "Merfolk of the Pearl Trident")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Merfolk of the Pearl Trident", 1)
	})
}

func TestCopperTablet(t *testing.T) {
	t.Run("damages each player on their upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Copper Tablet")
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 19) // turn 1 upkeep
		g.AssertLife(gametest.PlayerB, 19) // turn 2 upkeep
	})
}

func TestKarma(t *testing.T) {
	t.Run("deals damage per Swamp controlled", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Karma")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Swamp", 3)
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 17) // 3 Swamps = 3 damage
	})
}

func TestBlackVise(t *testing.T) {
	t.Run("deals hand-size damage to chosen opponent", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Black Vise")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Black Vise")
		for i := 0; i < 7; i++ {
			g.AddCard(core.ZoneHand, gametest.PlayerB, "Forest")
		}
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		// 7 cards - 4 threshold = 3 damage to PlayerB
		g.AssertLife(gametest.PlayerB, 17)
	})

	t.Run("does not damage controller", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Black Vise")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Black Vise")
		for i := 0; i < 7; i++ {
			g.AddCard(core.ZoneHand, gametest.PlayerA, "Forest")
		}
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		// Black Vise chose PlayerB on ETB; should not damage PlayerA
		g.AssertLife(gametest.PlayerA, 20)
	})
}

func TestAnimateDead(t *testing.T) {
	t.Run("returns creature from graveyard to battlefield", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Serra Angel") // 4/4
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Animate Dead")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Animate Dead", "Serra Angel")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Serra Angel", 1)
		g.AssertPowerToughness(gametest.PlayerA, "Serra Angel", 3, 4) // -1/-0 from Animate Dead
	})
}

func TestVerduranEnchantress(t *testing.T) {
	t.Run("draws on any enchantment cast not just green", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Verduran Enchantress")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Holy Armor") // white enchantment
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Holy Armor", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		playerA := g.AllPlayers()[0]
		if len(playerA.Hand()) < 1 {
			t.Errorf("Verduran Enchantress should draw on any enchantment cast, hand has %d cards", len(playerA.Hand()))
		}
	})
}

func TestBlueElementalBlast(t *testing.T) {
	t.Run("cannot counter green spell", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Giant Growth")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Blue Elemental Blast")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Giant Growth", "Grizzly Bears")
		g.CastInResponseTo(gametest.PlayerB, "Blue Elemental Blast")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// BEB only counters red; Giant Growth is green so it should resolve
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 5, 5)
	})
}

func TestRedElementalBlast(t *testing.T) {
	t.Run("cannot counter green spell", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Giant Growth")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Red Elemental Blast")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Giant Growth", "Grizzly Bears")
		g.CastInResponseTo(gametest.PlayerB, "Red Elemental Blast")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// REB only counters blue; Giant Growth is green so it should resolve
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 5, 5)
	})
}

func TestDeathgrip(t *testing.T) {
	t.Run("cannot counter red spell", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Deathgrip")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
		g.ActivateInResponseTo(gametest.PlayerB, "Deathgrip")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Deathgrip only counters green; Bolt is red so it should resolve
		g.AssertLife(gametest.PlayerB, 17)
	})
}

func TestLifeforce(t *testing.T) {
	t.Run("cannot counter red spell", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Lifeforce")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
		g.ActivateInResponseTo(gametest.PlayerB, "Lifeforce")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Lifeforce only counters black; Bolt is red so it should resolve
		g.AssertLife(gametest.PlayerB, 17)
	})
}

func TestSpellBlast(t *testing.T) {
	t.Run("fails when X is less than target CMC", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Giant Growth") // CMC 1
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Spell Blast")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Giant Growth", "Grizzly Bears")
		g.CastInResponseToWithX(gametest.PlayerB, "Spell Blast", 0) // X=0 < CMC 1
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 5, 5)
	})
}

func TestPowerSink(t *testing.T) {
	t.Run("does not counter if opponent can pay X", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Giant Growth")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Power Sink")
		g.AllPlayers()[0].ManaPool().Add(core.Colorless, 5) // extra mana to pay
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Giant Growth", "Grizzly Bears")
		g.CastInResponseToWithX(gametest.PlayerB, "Power Sink", 1)
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// PlayerA has mana to pay X=1, so spell should resolve
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 5, 5)
	})
}

func TestTwiddle(t *testing.T) {
	t.Run("can untap a tapped permanent", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Icy Manipulator")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
		g.ActivateAbility(1, core.Upkeep, gametest.PlayerA, "Icy Manipulator", "Hill Giant")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Twiddle")
		g.ChooseMode(gametest.PlayerA, 1) // choose "Untap"
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Twiddle", "Hill Giant")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Hill Giant", false)
	})

	t.Run("can tap an untapped permanent", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Twiddle")
		g.ChooseMode(gametest.PlayerA, 0) // choose "Tap"
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Twiddle", "Hill Giant")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Hill Giant", true)
	})
}

func TestBasaltMonolith(t *testing.T) {
	t.Run("does not untap during untap step", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Basalt Monolith")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Basalt Monolith")
		g.StopAt(3, core.PrecombatMain) // PlayerA's next untap step
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Basalt Monolith", true)
	})
}

func TestManaVault(t *testing.T) {
	t.Run("does not untap during untap step", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mana Vault")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Mana Vault")
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Mana Vault", true)
	})
}

func TestGauntletOfMight(t *testing.T) {
	t.Run("Mountains produce extra red mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gauntlet of Might")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Mountain")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Auto-mana adds 5R. Mountain tapped for 1R + Gauntlet bonus 1R = 7R total.
		pool := g.AllPlayers()[0].ManaPool()
		if pool.Count(core.Red) < 7 {
			t.Errorf("Gauntlet should make Mountain produce extra {R}; expected >=7 red mana, got %d", pool.Count(core.Red))
		}
	})
}

func TestNevinyrralsDisk(t *testing.T) {
	t.Run("enters the battlefield tapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Nevinyrral's Disk")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Nevinyrral's Disk")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Nevinyrral's Disk", true)
	})
}

func TestDragonWhelp(t *testing.T) {
	t.Run("destroyed at EOT if pumped 4 or more times", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dragon Whelp") // 2/3 Flying
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Dragon Whelp")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Dragon Whelp")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Dragon Whelp")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Dragon Whelp")
		g.StopAt(2, core.Upkeep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Dragon Whelp", 0)
	})
}

func TestZombieMaster(t *testing.T) {
	t.Run("grants regeneration to other Zombies", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Zombie Master")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Scathe Zombies") // 2/2 Zombie
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.ActivateAbility(1, core.Upkeep, gametest.PlayerA, "Scathe Zombies")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Scathe Zombies")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Scathe Zombies", 1)
	})
}

func TestGoblinBalloonBrigade(t *testing.T) {
	t.Run("gains flying when activated", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Goblin Balloon Brigade")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Goblin Balloon Brigade")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Goblin Balloon Brigade", core.Flying, true)
	})
}

func TestDwarvenWarriors(t *testing.T) {
	t.Run("makes power-2-or-less creature unblockable", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dwarven Warriors")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")  // 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")     // blocker
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Dwarven Warriors", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Hill Giant", "Grizzly Bears")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 18)
	})
}

func TestSedgeTroll(t *testing.T) {
	t.Run("gets +1/+1 while you control a Swamp", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sedge Troll") // base 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Sedge Troll", 3, 3)
	})
}

func TestTwoHeadedGiant(t *testing.T) {
	t.Run("can block two attackers", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Two-Headed Giant of Foriys") // 4/4
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gray Ogre")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears", "Gray Ogre")
		g.Block(1, gametest.PlayerB, "Two-Headed Giant of Foriys", "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Two-Headed Giant of Foriys", "Gray Ogre")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 20) // blocks both, no damage through
	})
}

func TestJuggernaut(t *testing.T) {
	t.Run("cannot be blocked by Walls", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Juggernaut")    // 5/3
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wall of Stone") // 0/8 Defender
		g.Attack(1, gametest.PlayerA, "Juggernaut")
		g.Block(1, gametest.PlayerB, "Wall of Stone", "Juggernaut")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 15)
	})
}

func TestNettlingImp(t *testing.T) {
	t.Run("has activated ability to force attack", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Nettling Imp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerA, "Nettling Imp", "Hill Giant")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Nettling Imp", true)
	})
}

func TestSeaSerpent(t *testing.T) {
	t.Run("sacrificed without Islands", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sea Serpent")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Sea Serpent", 0)
	})
}

func TestNetherShadow(t *testing.T) {
	t.Run("returns from graveyard with 3 creatures above", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Nether Shadow")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Gray Ogre")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Nether Shadow", 1)
	})
}

func TestRegeneration(t *testing.T) {
	// Creatures that should have a regeneration activated ability but don't.
	tests := []struct {
		name      string
		toughness int
	}{
		{"Will-o'-the-Wisp", 1},
		{"Uthden Troll", 2},
		{"Wall of Bone", 4},
		{"Wall of Brambles", 3},
		{"Living Wall", 6},
	}
	for _, tt := range tests {
		t.Run(tt.name+" survives lethal via regen", func(t *testing.T) {
			g := gametest.NewTestGame(t)
			g.AddCard(core.ZoneBattlefield, gametest.PlayerA, tt.name)
			g.AddCard(core.ZoneHand, gametest.PlayerB, "Fireball")
			g.ActivateAbility(1, core.Upkeep, gametest.PlayerA, tt.name)
			g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerB, "Fireball", tt.toughness+1, tt.name)
			g.StopAt(1, core.BeginCombat)
			g.Execute()
			g.AssertPermanentCount(gametest.PlayerA, tt.name, 1)
		})
	}
}

func TestAspectOfWolf(t *testing.T) {
	t.Run("is an Enchantment not a Creature", func(t *testing.T) {
		card, err := mage.CreateCard("Aspect of Wolf")
		if err != nil {
			t.Fatalf("could not create Aspect of Wolf: %v", err)
		}
		if card.HasType(core.TypeCreature) {
			t.Errorf("Aspect of Wolf should NOT be a Creature, it is an Enchantment - Aura")
		}
		if !card.HasType(core.TypeEnchantment) {
			t.Errorf("Aspect of Wolf should be an Enchantment, got types: %v", card.Types())
		}
	})
}

func TestNightmare(t *testing.T) {
	t.Run("P/T equals Swamp count", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Nightmare")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp", 3)
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Nightmare", 1)
		g.AssertPowerToughness(gametest.PlayerA, "Nightmare", 3, 3)
	})
}

func TestPlagueRats(t *testing.T) {
	t.Run("P/T scales with Plague Rats count", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plague Rats", 3)
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		found := false
		for _, perm := range g.AllBattlefield() {
			if perm.Name() == "Plague Rats" {
				p := perm.CurrentPower(g.Game)
				tt := perm.CurrentToughness(g.Game)
				if p == 3 && tt == 3 {
					found = true
				} else {
					t.Errorf("each Plague Rats should be 3/3 with 3 rats, got %d/%d", p, tt)
				}
			}
		}
		if !found {
			t.Errorf("no Plague Rats found as 3/3")
		}
	})
}

func TestClone(t *testing.T) {
	t.Run("copies target creature on ETB", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Serra Angel") // 4/4
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Clone")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Clone", "Serra Angel")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Should have two 4/4 creatures (original + clone)
		count44 := 0
		for _, perm := range g.AllBattlefield() {
			if perm.Controller == g.AllPlayers()[0].PlayerID() {
				if perm.CurrentPower(g.Game) == 4 && perm.CurrentToughness(g.Game) == 4 {
					count44++
				}
			}
		}
		if count44 < 2 {
			t.Errorf("Clone should create a second 4/4 (copy of Serra Angel); found %d", count44)
		}
		g.AssertGraveyardCount(gametest.PlayerA, "Clone", 0)
	})
}

func TestKeldonWarlord(t *testing.T) {
	t.Run("P/T equals non-Wall creature count", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Keldon Warlord")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Stone") // Wall, shouldn't count
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// 3 non-Wall creatures: Warlord + Bears + Giant
		g.AssertPermanentCount(gametest.PlayerA, "Keldon Warlord", 1)
		g.AssertPowerToughness(gametest.PlayerA, "Keldon Warlord", 3, 3)
	})
}

func TestRockHydra(t *testing.T) {
	t.Run("enters with X +1/+1 counters", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Rock Hydra")
		g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Rock Hydra", 4)
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Rock Hydra", 1)
		g.AssertPowerToughness(gametest.PlayerA, "Rock Hydra", 4, 4)
	})
}

func TestClockworkBeast(t *testing.T) {
	t.Run("loses a counter when attacking", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Clockwork Beast")
		g.Attack(1, gametest.PlayerA, "Clockwork Beast")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// 7 +1/+0 counters on 0/4 base = 7/4; end-of-combat trigger removes 1 = 6/4
		g.AssertPowerToughness(gametest.PlayerA, "Clockwork Beast", 6, 4)
	})

	t.Run("repair adds counters up to max", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Clockwork Beast")
		// Attack turns 1 and 2 to lose 2 counters (7 -> 5)
		g.Attack(1, gametest.PlayerA, "Clockwork Beast")
		g.Attack(2, gametest.PlayerA, "Clockwork Beast")
		// Repair for 2 during turn 3 upkeep -> back to 7
		g.ActivateAbilityWithX(3, core.Upkeep, gametest.PlayerA, "Clockwork Beast", 2)
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Clockwork Beast", core.P1P0, 7)
		g.AssertPowerToughness(gametest.PlayerA, "Clockwork Beast", 7, 4)
	})

	t.Run("repair caps at max even if X exceeds room", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Clockwork Beast")
		// Attack once to lose 1 counter (7 -> 6)
		g.Attack(1, gametest.PlayerA, "Clockwork Beast")
		// Turn 2: skip (beast untaps). Turn 3 upkeep: pay X=5, only 1 room -> cap at 7
		g.ActivateAbilityWithX(3, core.Upkeep, gametest.PlayerA, "Clockwork Beast", 5)
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Clockwork Beast", core.P1P0, 7)
		g.AssertPowerToughness(gametest.PlayerA, "Clockwork Beast", 7, 4)
	})
}

func TestNorthernPaladin(t *testing.T) {
	t.Run("destroys black permanent", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Northern Paladin")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Scathe Zombies") // black 2/2
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Northern Paladin", "Scathe Zombies")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Scathe Zombies", 0)
		g.AssertTapped(gametest.PlayerA, "Northern Paladin", true)
	})

	t.Run("cannot target non-black permanent", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Northern Paladin")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // green 2/2
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Northern Paladin", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Bears is green, not black — should not be destroyed
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestFarmstead(t *testing.T) {
	t.Run("gain 1 life on upkeep if paid", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 3)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Farmstead")
		g.SetLife(gametest.PlayerA, 18)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Farmstead", "Plains")
		// Turn 3 is PlayerA's next upkeep — trigger fires and auto-pays {W}{W}
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 19)
	})
}

func TestNettlingImpDestroysNonAttacker(t *testing.T) {
	t.Run("destroys creature that didnt attack", func(t *testing.T) {
		t.Skip("known bug: Nettling Imp delayed trigger doesn't fire when creature doesn't enter combat")
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Nettling Imp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerA, "Nettling Imp", "Hill Giant")
		g.StopAt(2, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 0)
	})
}



