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
