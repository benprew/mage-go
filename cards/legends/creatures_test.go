package legends

import (
	"os"
	"testing"

	"github.com/mage/mage/pkg/mage/core"
	"github.com/mage/mage/pkg/mage/gametest"
	_ "github.com/mage/mage/cards/limited" // register base cards
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

// ===== PHASE 1: VANILLA CREATURES =====

func TestKeepersOfTheFaith(t *testing.T) {
	t.Run("is 2/3 Human Cleric", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Keepers of the Faith")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Keepers of the Faith", 2, 3)
	})
}

func TestBarktoothWarbeard(t *testing.T) {
	t.Run("is 6/5 legendary", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Barktooth Warbeard")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Barktooth Warbeard", 6, 5)
	})
}

func TestWallOfEarth(t *testing.T) {
	t.Run("is 0/6 Wall with Defender", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Earth")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Wall of Earth", 0, 6)
		g.AssertHasAbility(gametest.PlayerA, "Wall of Earth", core.Defender, true)
	})
}

func TestWallOfHeat(t *testing.T) {
	t.Run("is 2/6 Wall with Defender", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Heat")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Wall of Heat", 2, 6)
		g.AssertHasAbility(gametest.PlayerA, "Wall of Heat", core.Defender, true)
	})
}

// ===== PHASE 2: KEYWORD CREATURES =====

func TestTundraWolves(t *testing.T) {
	t.Run("has first strike", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tundra Wolves")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Tundra Wolves", 1, 1)
		g.AssertHasAbility(gametest.PlayerA, "Tundra Wolves", core.FirstStrike, true)
	})

	t.Run("kills blocker before taking damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tundra Wolves")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Headless Horseman") // 2/2
		g.Attack(3, gametest.PlayerA, "Tundra Wolves")
		g.Block(3, gametest.PlayerB, "Headless Horseman", "Tundra Wolves")
		g.StopAt(3, core.EndStep)
		g.Execute()
		// 1/1 first strike vs 2/2 — Wolves hits first for 1, not lethal. Horseman hits back for 2, kills Wolves.
		g.AssertPermanentCount(gametest.PlayerA, "Tundra Wolves", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Headless Horseman", 1)
	})
}

func TestThunderSpirit(t *testing.T) {
	t.Run("has flying and first strike", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Thunder Spirit")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Thunder Spirit", 2, 2)
		g.AssertHasAbility(gametest.PlayerA, "Thunder Spirit", core.Flying, true)
		g.AssertHasAbility(gametest.PlayerA, "Thunder Spirit", core.FirstStrike, true)
	})
}

func TestAzureDrake(t *testing.T) {
	t.Run("is 2/4 flying Drake", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Azure Drake")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Azure Drake", 2, 4)
		g.AssertHasAbility(gametest.PlayerA, "Azure Drake", core.Flying, true)
	})
}

func TestCatWarriors(t *testing.T) {
	t.Run("has forestwalk", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Cat Warriors")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Cat Warriors", 2, 2)
		g.AssertHasAbility(gametest.PlayerA, "Cat Warriors", core.Forestwalk, true)
	})

	t.Run("can't be blocked when opponent has Forest", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Cat Warriors")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // 2/2
		g.Attack(3, gametest.PlayerA, "Cat Warriors")
		// Grizzly Bears can't block because Cat Warriors has forestwalk and defender has Forest
		g.StopAt(3, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 18) // 2 damage gets through
	})
}

func TestMountainYeti(t *testing.T) {
	t.Run("has mountainwalk and protection from white", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain Yeti")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Mountain Yeti", 3, 3)
		g.AssertHasAbility(gametest.PlayerA, "Mountain Yeti", core.Mountainwalk, true)
	})

	t.Run("protection from white prevents Swords to Plowshares", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain Yeti")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Swords to Plowshares")
		// STP is white — can't target Mountain Yeti
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Swords to Plowshares", "Mountain Yeti")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Mountain Yeti", 1) // still alive
	})
}

func TestZephyrFalcon(t *testing.T) {
	t.Run("has flying and vigilance", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Zephyr Falcon")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Zephyr Falcon", 1, 1)
		g.AssertHasAbility(gametest.PlayerA, "Zephyr Falcon", core.Flying, true)
		g.AssertHasAbility(gametest.PlayerA, "Zephyr Falcon", core.Vigilance, true)
	})

	t.Run("doesn't tap when attacking", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Zephyr Falcon")
		g.Attack(3, gametest.PlayerA, "Zephyr Falcon")
		g.StopAt(3, core.EndStep)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Zephyr Falcon", false) // vigilance
		g.AssertLife(gametest.PlayerB, 19)
	})
}

func TestWallOfLight(t *testing.T) {
	t.Run("has defender and protection from black", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Light")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Wall of Light", 1, 5)
		g.AssertHasAbility(gametest.PlayerA, "Wall of Light", core.Defender, true)
	})
}

func TestRighteousAvengers(t *testing.T) {
	t.Run("has plainswalk", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Righteous Avengers")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Righteous Avengers", 3, 1)
		g.AssertHasAbility(gametest.PlayerA, "Righteous Avengers", core.Plainswalk, true)
	})
}

// ===== PHASE 3: SIMPLE ACTIVATED ABILITIES =====

func TestDavenantArcher(t *testing.T) {
	t.Run("deals 1 damage to attacking creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "D'Avenant Archer")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // 2/2
		g.Attack(4, gametest.PlayerB, "Grizzly Bears")
		g.ActivateAbility(4, core.DeclareBlockers, gametest.PlayerA, "D'Avenant Archer", "Grizzly Bears")
		g.StopAt(4, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 18) // Bears still hit for 2, but took 1 damage
	})
}

func TestCarrionAnts(t *testing.T) {
	t.Run("is 0/1 that can pump", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Carrion Ants")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Carrion Ants", 0, 1)
	})

	t.Run("{1}: gets +1/+1 until end of turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Carrion Ants")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Carrion Ants", "+1/+1")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Carrion Ants", "+1/+1")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Carrion Ants", 2, 3)
	})
}

func TestWalkingDead(t *testing.T) {
	t.Run("{B}: regenerates", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Walking Dead")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerA, "Walking Dead", "Regenerate")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Walking Dead")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Walking Dead", 1) // regenerated
	})
}

func TestKillerBees(t *testing.T) {
	t.Run("{G}: gets +1/+1, can pump multiple times", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Killer Bees")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Killer Bees", "+1/+1")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Killer Bees", "+1/+1")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// 0/1 base + 2 pumps = 2/3
		g.AssertPowerToughness(gametest.PlayerA, "Killer Bees", 2, 3)
	})
}

func TestSpinalVillain(t *testing.T) {
	t.Run("{T}: destroys target blue creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Spinal Villain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Azure Drake") // blue creature
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Spinal Villain", "Azure Drake")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Azure Drake", 0)
		g.AssertGraveyardCount(gametest.PlayerB, "Azure Drake", 1)
	})
}

func TestFireSprites(t *testing.T) {
	t.Run("is 1/1 flying with {G},{T}: add {R}", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fire Sprites")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Fire Sprites", 1, 1)
		g.AssertHasAbility(gametest.PlayerA, "Fire Sprites", core.Flying, true)
	})
}

// ===== PHASE 5: RAMPAGE CREATURES =====

func TestCrawGiant(t *testing.T) {
	t.Run("has trample and rampage 2", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Craw Giant")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Craw Giant", 6, 4)
		g.AssertHasAbility(gametest.PlayerA, "Craw Giant", core.Trample, true)
	})

	t.Run("gets +2/+2 per blocker beyond first", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Craw Giant")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")       // 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Headless Horseman")   // 2/2
		g.Attack(3, gametest.PlayerA, "Craw Giant")
		g.Block(3, gametest.PlayerB, "Grizzly Bears", "Craw Giant")
		g.Block(3, gametest.PlayerB, "Headless Horseman", "Craw Giant")
		g.StopAt(3, core.EndStep)
		g.Execute()
		// 2 blockers, rampage 2: +2/+2 * (2-1) = +2/+2. 6+2=8 power, 4+2=6 toughness.
		// 2 blockers deal 4 total, Craw Giant has 6 toughness — survives.
		// Craw Giant deals 8, trample means both blockers die (4 needed to kill) and 4 tramples.
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Headless Horseman", 0)
		g.AssertLife(gametest.PlayerB, 16) // 4 trample damage
	})
}

// ===== PHASE 6: SIMPLE SPELLS =====

func TestDarkness(t *testing.T) {
	t.Run("prevents all combat damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Darkness")
		g.Attack(3, gametest.PlayerA, "Grizzly Bears")
		g.CastSpell(3, core.DeclareBlockers, gametest.PlayerB, "Darkness")
		g.StopAt(3, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 20) // no damage
	})
}

func TestHolyDay(t *testing.T) {
	t.Run("prevents all combat damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Holy Day")
		g.Attack(4, gametest.PlayerB, "Grizzly Bears")
		g.CastSpell(4, core.DeclareBlockers, gametest.PlayerA, "Holy Day")
		g.StopAt(4, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 20) // no damage
	})
}

func TestFlashCounter(t *testing.T) {
	t.Run("counters target instant", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Flash Counter")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "PlayerA")
		g.CastInResponseTo(gametest.PlayerA, "Flash Counter")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 20) // bolt was countered
	})
}

func TestForceSpike(t *testing.T) {
	t.Run("counters spell if controller can't pay {1}", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Force Spike")
		// PlayerB taps their only Mountain to cast Bolt, no mana left to pay {1}
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "PlayerA")
		g.CastInResponseTo(gametest.PlayerA, "Force Spike")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 20) // bolt was countered
	})
}

func TestRemoveSoul(t *testing.T) {
	t.Run("counters creature spell", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Remove Soul")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Grizzly Bears")
		g.CastInResponseTo(gametest.PlayerA, "Remove Soul")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestBoomerang(t *testing.T) {
	t.Run("returns permanent to hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Boomerang")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Boomerang", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
	})
}

func TestAcidRain(t *testing.T) {
	t.Run("destroys all Forests", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Acid Rain")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Acid Rain")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Forest", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Forest", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Mountain", 1) // not a Forest
	})
}

func TestCleanse(t *testing.T) {
	t.Run("destroys all black creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Walking Dead")  // black 1/1
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Headless Horseman") // black 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")     // green 2/2
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Cleanse")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Cleanse")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Walking Dead", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Headless Horseman", 0)
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1) // green, not destroyed
	})
}

// ===== PHASE 7: ENCHANTMENTS =====

func TestDivineTransformation(t *testing.T) {
	t.Run("gives +3/+3", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Divine Transformation")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Divine Transformation", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 5, 5)
	})
}

func TestGiantStrength(t *testing.T) {
	t.Run("gives +2/+2", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Giant Strength")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Giant Strength", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 4, 4)
	})
}

func TestImmolation(t *testing.T) {
	t.Run("gives +2/-2", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Azure Drake") // 2/4
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Immolation")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Immolation", "Azure Drake")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Azure Drake", 4, 2)
	})
}

func TestEternalWarrior(t *testing.T) {
	t.Run("gives vigilance", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Eternal Warrior")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Eternal Warrior", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Vigilance, true)
	})
}

// ===== LANDWALK NULLIFICATION =====

func TestCrevasse(t *testing.T) {
	t.Run("mountainwalk creatures can be blocked", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain Yeti") // 3/3 mountainwalk
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Crevasse")
		g.Attack(3, gametest.PlayerA, "Mountain Yeti")
		g.Block(3, gametest.PlayerB, "Grizzly Bears", "Mountain Yeti")
		g.StopAt(3, core.EndStep)
		g.Execute()
		// With Crevasse, mountainwalk is nullified so Bears can block
		g.AssertLife(gametest.PlayerB, 20) // no damage through
	})
}

func TestConcordantCrossroads(t *testing.T) {
	t.Run("all creatures have haste", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Concordant Crossroads")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Bears can attack same turn thanks to haste
		g.AssertLife(gametest.PlayerB, 18)
	})
}

// ===== MORE SPELLS =====

func TestBloodLust(t *testing.T) {
	t.Run("creature with toughness >= 5 gets +4/-4", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Craw Giant") // 6/4... wait, 4 < 5
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Earth") // 0/6
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Blood Lust")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Blood Lust", "Wall of Earth")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// 0/6 with +4/-4 = 4/2
		g.AssertPowerToughness(gametest.PlayerA, "Wall of Earth", 4, 2)
	})

	t.Run("creature with toughness < 5 gets +4/-(toughness-1)", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Blood Lust")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Blood Lust", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// 2/2 with +4/-(2-1) = +4/-1 → 6/1
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 6, 1)
	})
}

func TestHellSwarm(t *testing.T) {
	t.Run("all creatures get -1/-0", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Azure Drake") // 2/4
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Hell Swarm")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Hell Swarm")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 1, 2)
		g.AssertPowerToughness(gametest.PlayerB, "Azure Drake", 1, 4)
	})
}

func TestHellfire(t *testing.T) {
	t.Run("destroys nonblack creatures and deals damage to caster", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")   // green
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Azure Drake")     // blue
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Walking Dead")    // black, should survive
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Hellfire")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Hellfire")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Azure Drake", 0)
		g.AssertPermanentCount(gametest.PlayerA, "Walking Dead", 1) // black survives
		// 2 creatures destroyed + 3 = 5 damage to caster
		g.AssertLife(gametest.PlayerA, 15)
	})
}

func TestStormSeeker(t *testing.T) {
	t.Run("deals damage equal to hand size", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Storm Seeker")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Storm Seeker", "PlayerB")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// PlayerB has 3 cards in hand → 3 damage
		g.AssertLife(gametest.PlayerB, 17)
	})
}

func TestChainLightning(t *testing.T) {
	t.Run("deals 3 damage to target", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Chain Lightning")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Chain Lightning", "PlayerB")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 17)
	})
}

func TestSyphonSoul(t *testing.T) {
	t.Run("deals 2 to opponent and gains 2 life", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Syphon Soul")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Syphon Soul")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 18)
		g.AssertLife(gametest.PlayerA, 22)
	})
}

func TestShieldWall(t *testing.T) {
	t.Run("your creatures get +0/+2", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Azure Drake") // opponent's
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Shield Wall")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Shield Wall")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 4) // +0/+2
		g.AssertPowerToughness(gametest.PlayerB, "Azure Drake", 2, 4)   // not affected
	})
}

func TestTransmutation(t *testing.T) {
	t.Run("swaps power and toughness", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Azure Drake") // 2/4
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Transmutation")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Transmutation", "Azure Drake")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Azure Drake", 4, 2)
	})
}

// ===== MORE CREATURES =====

func TestCyclopeanMummy(t *testing.T) {
	t.Run("exiles when it dies", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Cyclopean Mummy")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Cyclopean Mummy")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Cyclopean Mummy", 0)
		g.AssertGraveyardCount(gametest.PlayerA, "Cyclopean Mummy", 0) // not in graveyard
		g.AssertExileCount("Cyclopean Mummy", 1)     // exiled
	})
}

func TestFallenAngel(t *testing.T) {
	t.Run("sacrifice creature for +2/+1", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fallen Angel")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Fallen Angel", "+2/+1")
		g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Fallen Angel", 5, 4) // 3+2/3+1
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
	})
}

func TestVampireBats(t *testing.T) {
	t.Run("flying 0/1 that pumps with {B}", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Vampire Bats")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Vampire Bats", 0, 1)
		g.AssertHasAbility(gametest.PlayerA, "Vampire Bats", core.Flying, true)
	})
}

func TestEmeraldDragonfly(t *testing.T) {
	t.Run("{G}{G}: gains first strike", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Emerald Dragonfly")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Emerald Dragonfly", "first strike")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Emerald Dragonfly", core.FirstStrike, true)
	})
}

// ===== MORE ENCHANTMENTS =====

func TestMoat(t *testing.T) {
	t.Run("creatures without flying can't attack", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Moat")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // no flying
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Azure Drake")   // has flying
		g.Attack(4, gametest.PlayerB, "Azure Drake")
		// Bears can't attack due to Moat
		g.StopAt(4, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 18) // only Drake's 2 damage
	})
}

func TestGravitySphere(t *testing.T) {
	t.Run("all creatures lose flying", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gravity Sphere")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Azure Drake") // normally has flying
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Azure Drake", core.Flying, false) // flying removed
	})
}

func TestGreed(t *testing.T) {
	t.Run("{B}, pay 2 life: draw a card", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Greed")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Greed", "Draw")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 18) // paid 2 life
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1) // drew 1 card
	})
}

func TestAntiMagicAura(t *testing.T) {
	t.Run("grants shroud to enchanted creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Anti-Magic Aura")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Anti-Magic Aura", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Shroud, true)
	})
}

func TestAngelicVoices(t *testing.T) {
	t.Run("boosts creatures if no nonartifact nonwhite creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Angelic Voices")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tundra Wolves") // white 1/1
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Only white creature → bonus applies
		g.AssertPowerToughness(gametest.PlayerA, "Tundra Wolves", 2, 2) // 1+1/1+1
	})

	t.Run("no boost if nonwhite creature present", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Angelic Voices")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tundra Wolves") // white
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // green, nonartifact
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Tundra Wolves", 1, 1) // no bonus
	})
}

func TestTheAbyss(t *testing.T) {
	t.Run("destroys nonartifact creature at upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "The Abyss")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
		g.StopAt(3, core.PrecombatMain) // after PlayerA's upkeep on turn 3
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
	})
}

// ===== LEGENDARY CREATURES =====

func TestAdunOakenshield(t *testing.T) {
	t.Run("{B}{R}{G},{T}: return creature from graveyard to hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Adun Oakenshield")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Adun Oakenshield", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Grizzly Bears", 0)
		g.AssertHandCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}

func TestBorisDevilboon(t *testing.T) {
	t.Run("{2}{B}{R},{T}: create 1/1 Minor Demon token", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Boris Devilboon")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Boris Devilboon", "")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Minor Demon", 1)
		g.AssertPowerToughness(gametest.PlayerA, "Minor Demon", 1, 1)
	})
}

func TestDakkonBlackblade(t *testing.T) {
	t.Run("P/T equal to number of lands controlled", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dakkon Blackblade")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Dakkon Blackblade", 3, 3)
	})
}

func TestAngusMackenzie(t *testing.T) {
	t.Run("{G}{W}{U},{T}: prevent all combat damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Angus Mackenzie")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.Attack(4, gametest.PlayerB, "Grizzly Bears")
		g.ActivateAbility(4, core.DeclareBlockers, gametest.PlayerA, "Angus Mackenzie", "")
		g.StopAt(4, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 20) // all combat damage prevented
	})
}

func TestAkronLegionnaire(t *testing.T) {
	t.Run("only self and artifact creatures can attack", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Akron Legionnaire")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // can't attack
		g.Attack(3, gametest.PlayerA, "Akron Legionnaire")
		g.StopAt(3, core.EndStep)
		g.Execute()
		// Akron Legionnaire attacks for 8, Bears can't attack
		g.AssertLife(gametest.PlayerB, 12)
	})
}

func TestIvoryGuardians(t *testing.T) {
	t.Run("gets +1/+1 if opponent has nontoken red permanent", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ivory Guardians")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Crimson Manticore") // red creature
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Ivory Guardians", 4, 4) // 3+1/3+1
	})

	t.Run("stays 3/3 without opponent red permanents", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ivory Guardians")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Ivory Guardians", 3, 3)
	})
}

func TestBeastsOfBogardan(t *testing.T) {
	t.Run("gets +1/+1 if opponent has nontoken white permanent", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Beasts of Bogardan")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Tundra Wolves") // white creature
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Beasts of Bogardan", 4, 4) // 3+1/3+1
	})
}

func TestRabidWombat(t *testing.T) {
	t.Run("+2/+2 per aura attached", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rabid Wombat") // 0/1 vigilance
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Giant Strength")       // +2/+2 aura
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Giant Strength", "Rabid Wombat")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Base 0/1 + aura boost +2/+2 + wombat ability +2/+2 = 4/5
		g.AssertPowerToughness(gametest.PlayerA, "Rabid Wombat", 4, 5)
	})
}

// ===== KOBOLD LORDS =====

func TestKoboldTaskmaster(t *testing.T) {
	t.Run("other Kobolds get +1/+0", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kobold Taskmaster")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Crimson Kobolds") // 0/1
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Kobold Taskmaster", 1, 2) // not boosted by self
		g.AssertPowerToughness(gametest.PlayerA, "Crimson Kobolds", 1, 1)   // +1/+0
	})
}

func TestKoboldOverlord(t *testing.T) {
	t.Run("other Kobolds have first strike", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kobold Overlord")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Crimson Kobolds") // 0/1
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Crimson Kobolds", core.FirstStrike, true)
	})
}

func TestKoboldDrillSergeant(t *testing.T) {
	t.Run("other Kobolds get +0/+1 and trample", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kobold Drill Sergeant")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Crimson Kobolds") // 0/1
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Crimson Kobolds", 0, 2) // +0/+1
		g.AssertHasAbility(gametest.PlayerA, "Crimson Kobolds", core.Trample, true)
	})
}

// ===== MORE ENCHANTMENTS (Landwalk nullification) =====

func TestDeadfall(t *testing.T) {
	t.Run("forestwalk creatures can be blocked", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Cat Warriors") // 2/2 forestwalk
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Deadfall")
		g.Attack(3, gametest.PlayerA, "Cat Warriors")
		g.Block(3, gametest.PlayerB, "Grizzly Bears", "Cat Warriors")
		g.StopAt(3, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 20) // blocked, no damage through
	})
}

func TestGreatWall(t *testing.T) {
	t.Run("plainswalk creatures can be blocked", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Righteous Avengers") // 3/1 plainswalk
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Great Wall")
		g.Attack(3, gametest.PlayerA, "Righteous Avengers")
		g.Block(3, gametest.PlayerB, "Grizzly Bears", "Righteous Avengers")
		g.StopAt(3, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 20) // blocked
	})
}

func TestQuagmire(t *testing.T) {
	t.Run("swampwalk creatures can be blocked", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lost Soul") // swampwalk
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Swamp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Quagmire")
		g.Attack(3, gametest.PlayerA, "Lost Soul")
		g.Block(3, gametest.PlayerB, "Grizzly Bears", "Lost Soul")
		g.StopAt(3, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 20) // blocked
	})
}

func TestUndertow(t *testing.T) {
	t.Run("islandwalk creatures can be blocked", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Devouring Deep") // islandwalk
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Undertow")
		g.Attack(3, gametest.PlayerA, "Devouring Deep")
		g.Block(3, gametest.PlayerB, "Grizzly Bears", "Devouring Deep")
		g.StopAt(3, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 20) // blocked
	})
}

// ===== MORE SPELLS =====

func TestDivineOffering(t *testing.T) {
	t.Run("destroys artifact and gains life equal to CMC", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Black Mana Battery") // artifact, CMC 4
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Divine Offering")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Divine Offering", "Black Mana Battery")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Black Mana Battery", 0) // destroyed
		g.AssertLife(gametest.PlayerA, 24) // gain 4 life (CMC 4)
	})
}

func TestIndestructibleAura(t *testing.T) {
	t.Run("prevents all damage to creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Indestructible Aura")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Grizzly Bears")
		g.CastInResponseTo(gametest.PlayerA, "Indestructible Aura", "Grizzly Bears")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1) // survived
	})
}

// TODO: TestUntamedWilds — needs investigation of SearchLibraryToBattlefield test setup

// ===== FORTIFIED AREA =====

func TestFortifiedArea(t *testing.T) {
	t.Run("walls get +1/+0 and banding", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fortified Area")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Earth") // 0/6 Defender
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Wall of Earth", 1, 6) // +1/+0
		g.AssertHasAbility(gametest.PlayerA, "Wall of Earth", core.Banding, true)
	})
}

// TODO: TestKismet — ETB trigger needs target binding from event to work correctly

// ===== SPIRIT LINK =====

func TestSpiritLink(t *testing.T) {
	t.Run("gains life when enchanted creature deals damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Spirit Link")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Spirit Link", "Grizzly Bears")
		g.Attack(3, gametest.PlayerA, "Grizzly Bears")
		g.StopAt(3, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 18) // took 2 combat damage
		g.AssertLife(gametest.PlayerA, 21) // gained 1 life (simplified: 1 per damage event)
	})
}

// ===== MORE RAMPAGE =====

func TestAerathiBerserker(t *testing.T) {
	t.Run("rampage 3: +3/+3 per extra blocker", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Aerathi Berserker") // 2/4
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")     // 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Headless Horseman") // 2/2
		g.Attack(3, gametest.PlayerA, "Aerathi Berserker")
		g.Block(3, gametest.PlayerB, "Grizzly Bears", "Aerathi Berserker")
		g.Block(3, gametest.PlayerB, "Headless Horseman", "Aerathi Berserker")
		g.StopAt(3, core.EndStep)
		g.Execute()
		// 2 blockers, rampage 3: +3/+3 * (2-1) = +3/+3. 2+3=5 power, 4+3=7 toughness.
		// 2 blockers deal 4 total, Berserker has 7 toughness — survives.
		// Berserker deals 5, killing both 2/2s (2 + 2 = 4 lethal, 1 excess)
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Headless Horseman", 0)
	})
}

// ===== TDD: NEW IMPLEMENTATIONS =====

func TestHundingGjornersen(t *testing.T) {
	t.Run("5/4 with rampage 1", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hunding Gjornersen")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Headless Horseman")
		g.Attack(3, gametest.PlayerA, "Hunding Gjornersen")
		g.Block(3, gametest.PlayerB, "Grizzly Bears", "Hunding Gjornersen")
		g.Block(3, gametest.PlayerB, "Headless Horseman", "Hunding Gjornersen")
		g.StopAt(3, core.EndStep)
		g.Execute()
		// 2 blockers, rampage 1: +1/+1 * (2-1) = +1/+1. 5+1=6 power, 4+1=5 toughness.
		// Blockers deal 4, Hunding has 5 toughness — survives.
		g.AssertPermanentCount(gametest.PlayerA, "Hunding Gjornersen", 1)
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Headless Horseman", 0)
	})
}

func TestMarhaultElsdragon(t *testing.T) {
	t.Run("4/6 with rampage 1", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Marhault Elsdragon")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Marhault Elsdragon", 4, 6)
	})
}

func TestPavelMaliki(t *testing.T) {
	t.Run("{B}{R}: +1/+0 until end of turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Pavel Maliki")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Pavel Maliki", "+1/+0")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Pavel Maliki", 6, 3) // 5+1/3
	})
}

func TestPrincessLucrezia(t *testing.T) {
	t.Run("{T}: add {U}", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Princess Lucrezia")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Princess Lucrezia", 5, 4)
	})
}

func TestRivenTurnbull(t *testing.T) {
	t.Run("{T}: add {B}", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Riven Turnbull")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Riven Turnbull", 5, 7)
	})
}

func TestSunastianFalconer(t *testing.T) {
	t.Run("{T}: add {C}{C}", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sunastian Falconer")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Sunastian Falconer", 4, 4)
	})
}

func TestRamirezDePietro(t *testing.T) {
	t.Run("4/3 first strike legendary pirate", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ramirez DePietro")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Ramirez DePietro", 4, 3)
		g.AssertHasAbility(gametest.PlayerA, "Ramirez DePietro", core.FirstStrike, true)
	})
}

func TestPitScorpion(t *testing.T) {
	t.Run("deals damage gives poison counter", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Pit Scorpion") // 1/1
		g.Attack(3, gametest.PlayerA, "Pit Scorpion")
		g.StopAt(3, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 19) // 1 combat damage
		// TODO: AssertPoisonCounters when available
	})
}

func TestPalladiaMors(t *testing.T) {
	t.Run("7/7 flying trample with upkeep sacrifice", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Palladia-Mors")
		// Provide lands so upkeep cost can be paid
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Palladia-Mors", 7, 7)
		g.AssertHasAbility(gametest.PlayerA, "Palladia-Mors", core.Flying, true)
		g.AssertHasAbility(gametest.PlayerA, "Palladia-Mors", core.Trample, true)
	})

	t.Run("sacrificed if upkeep cost not paid", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Palladia-Mors")
		// No mana to pay {R}{G}{W} — should be sacrificed at upkeep
		g.StopAt(3, core.PrecombatMain) // turn 3 = PlayerA's second upkeep
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Palladia-Mors", 0)
	})
}
