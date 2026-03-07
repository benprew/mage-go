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
	t.Run("gains life equal to damage dealt by enchanted creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Spirit Link")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Spirit Link", "Grizzly Bears")
		g.Attack(3, gametest.PlayerA, "Grizzly Bears")
		g.StopAt(3, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 18) // took 2 combat damage
		g.AssertLife(gametest.PlayerA, 22) // gained 2 life (equal to damage dealt)
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

// ===== BATCH 5: TDD for legendary creatures =====

func TestXiraArien(t *testing.T) {
	t.Run("flying 1/2 with tap draw ability", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Xira Arien")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		// Put a known card on top of PlayerB's library
		g.AddCard(core.ZoneLibrary, gametest.PlayerB, "Grizzly Bears")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Xira Arien", "PlayerB")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Xira Arien", 1, 2)
		g.AssertHasAbility(gametest.PlayerA, "Xira Arien", core.Flying, true)
		// PlayerB drew the Grizzly Bears
		g.AssertHandCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestTuknirDeathlock(t *testing.T) {
	t.Run("flying 2/2 with tap boost ability", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tuknir Deathlock")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Tuknir Deathlock", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Tuknir Deathlock", 2, 2)
		g.AssertHasAbility(gametest.PlayerA, "Tuknir Deathlock", core.Flying, true)
		// Grizzly Bears should be 4/4 (2+2 / 2+2)
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 4, 4)
	})
}

func TestTorWauki(t *testing.T) {
	t.Run("tap to deal 2 to attacking or blocking creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tor Wauki")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		// PlayerB attacks with Grizzly Bears
		g.Attack(2, gametest.PlayerB, "Grizzly Bears")
		// PlayerA activates Tor Wauki to deal 2 to attacking Grizzly Bears
		g.ActivateAbility(2, core.DeclareBlockers, gametest.PlayerA, "Tor Wauki", "Grizzly Bears")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// 2 damage kills 2/2 Grizzly Bears
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestLadyCaleria(t *testing.T) {
	t.Run("tap to deal 3 to attacking or blocking creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lady Caleria")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Headless Horseman")
		g.Attack(2, gametest.PlayerB, "Headless Horseman")
		g.ActivateAbility(2, core.DeclareBlockers, gametest.PlayerA, "Lady Caleria", "Headless Horseman")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// 3 damage kills 2/2 Headless Horseman
		g.AssertGraveyardCount(gametest.PlayerB, "Headless Horseman", 1)
	})
}

func TestRagnar(t *testing.T) {
	t.Run("tap to regenerate target creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ragnar")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		// Give Bears a regen shield
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Ragnar", "Grizzly Bears")
		// Cast Lightning Bolt to deal 3 to Bears (enough to kill 2/2)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Bears should survive thanks to regeneration
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}

func TestSolkanarTheSwampKing(t *testing.T) {
	t.Run("5/5 swampwalk with life gain on black spell", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sol'kanar the Swamp King")
		// Cast a black spell to trigger life gain
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Dark Ritual")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Dark Ritual")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Sol'kanar the Swamp King", 5, 5)
		g.AssertHasAbility(gametest.PlayerA, "Sol'kanar the Swamp King", core.Swampwalk, true)
		// Gained 1 life from casting black spell
		g.AssertLife(gametest.PlayerA, 21)
	})
}

func TestNicolBolas(t *testing.T) {
	t.Run("7/7 flying elder dragon with upkeep sacrifice", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Nicol Bolas")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Nicol Bolas", 7, 7)
		g.AssertHasAbility(gametest.PlayerA, "Nicol Bolas", core.Flying, true)
	})

	t.Run("damage to opponent discards their hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Nicol Bolas")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		// Give PlayerB some cards in hand (use distinct creature names)
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Headless Horseman")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Raging Bull")
		g.Attack(1, gametest.PlayerA, "Nicol Bolas")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// PlayerB took 7 damage and discarded entire hand (3 cards → graveyard)
		g.AssertLife(gametest.PlayerB, 13)
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
		g.AssertGraveyardCount(gametest.PlayerB, "Headless Horseman", 1)
		g.AssertGraveyardCount(gametest.PlayerB, "Raging Bull", 1)
	})

	t.Run("sacrificed if upkeep cost not paid", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Nicol Bolas")
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Nicol Bolas", 0)
	})
}

func TestVaevictisAsmadi(t *testing.T) {
	t.Run("7/7 flying elder dragon with three pump abilities", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Vaevictis Asmadi")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		// Activate pump ability three times (each activation is +1/+0)
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Vaevictis Asmadi")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Vaevictis Asmadi")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Vaevictis Asmadi")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Vaevictis Asmadi", 10, 7)
		g.AssertHasAbility(gametest.PlayerA, "Vaevictis Asmadi", core.Flying, true)
	})

	t.Run("sacrificed if upkeep cost not paid", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Vaevictis Asmadi")
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Vaevictis Asmadi", 0)
	})
}

func TestLordMagnus(t *testing.T) {
	t.Run("4/3 first strike nullifies plainswalk and forestwalk", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lord Magnus")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		// Cat Warriors have forestwalk — normally unblockable if opponent has Forest
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Cat Warriors")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		// Lord Magnus nullifies forestwalk
		g.Attack(2, gametest.PlayerB, "Cat Warriors")
		g.Block(2, gametest.PlayerA, "Grizzly Bears", "Cat Warriors")
		g.StopAt(2, core.EndStep)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Lord Magnus", 4, 3)
		g.AssertHasAbility(gametest.PlayerA, "Lord Magnus", core.FirstStrike, true)
		// Cat Warriors was blocked (forestwalk nullified)
		g.AssertLife(gametest.PlayerA, 20)
	})
}

func TestUrDrago(t *testing.T) {
	t.Run("4/4 first strike nullifies swampwalk", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ur-Drago")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		// Lost Soul has swampwalk
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Lost Soul")
		g.Attack(2, gametest.PlayerB, "Lost Soul")
		g.Block(2, gametest.PlayerA, "Grizzly Bears", "Lost Soul")
		g.StopAt(2, core.EndStep)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Ur-Drago", 4, 4)
		g.AssertHasAbility(gametest.PlayerA, "Ur-Drago", core.FirstStrike, true)
		// Lost Soul was blocked (swampwalk nullified)
		g.AssertLife(gametest.PlayerA, 20)
	})
}

func TestJacquesLeVert(t *testing.T) {
	t.Run("green creatures you control get +0/+2", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Jacques le Vert")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		// Non-green creature should NOT get boost
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Headless Horseman")
		// Opponent's green creature should NOT get boost
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Cat Warriors")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Jacques le Vert is 3/2 — he has red, green, and white colors. He IS green, so he gets +0/+2 = 3/4
		g.AssertPowerToughness(gametest.PlayerA, "Jacques le Vert", 3, 4)
		// Grizzly Bears is 2/2 green → 2/4
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 4)
		// Headless Horseman is 2/2 black → no boost
		g.AssertPowerToughness(gametest.PlayerA, "Headless Horseman", 2, 2)
		// Opponent's Cat Warriors → no boost (not controlled by you)
		g.AssertPowerToughness(gametest.PlayerB, "Cat Warriors", 2, 2)
	})
}

func TestKeiTakahashi(t *testing.T) {
	t.Run("tap to prevent next 2 damage to creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kei Takahashi")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		// Prevent 2 damage to Bears
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Kei Takahashi", "Grizzly Bears")
		// Lightning Bolt deals 3 to Bears; 2 prevented, 1 gets through
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Bears took 1 damage (3 - 2 prevention = 1) — still alive (2 toughness - 1 = survives)
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}

func TestRamsesOverdark(t *testing.T) {
	t.Run("destroys enchanted creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ramses Overdark")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		// Cast Holy Strength on Bears to make them "enchanted"
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Holy Strength")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Holy Strength", "Grizzly Bears")
		// Then use Ramses to destroy the enchanted creature
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Ramses Overdark", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Grizzly Bears should be destroyed
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	})

	t.Run("cannot target non-enchanted creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ramses Overdark")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		// No aura → not enchanted → ability should fail to find valid target
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Ramses Overdark", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Bears still alive since they are not a valid target
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestTetsuoUmezawa(t *testing.T) {
	t.Run("destroys tapped creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tetsuo Umezawa")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		// Bears attack on turn 2 (PlayerB's turn), becoming tapped
		g.Attack(2, gametest.PlayerB, "Grizzly Bears")
		// Tetsuo destroys tapped Bears after blockers
		g.ActivateAbility(2, core.DeclareBlockers, gametest.PlayerA, "Tetsuo Umezawa", "Grizzly Bears")
		g.StopAt(2, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	})

	t.Run("cannot target untapped non-blocking creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tetsuo Umezawa")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		// Bears are untapped and not blocking → invalid target
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Tetsuo Umezawa", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestAmrouKithkin(t *testing.T) {
	t.Run("cannot be blocked by power 3 or greater", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Amrou Kithkin")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Craw Wurm") // 6/4, power >= 3
		g.Attack(1, gametest.PlayerA, "Amrou Kithkin")
		g.Block(1, gametest.PlayerB, "Craw Wurm", "Amrou Kithkin")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Craw Wurm can't block Amrou Kithkin — 1 damage to player B
		g.AssertLife(gametest.PlayerB, 19)
	})

	t.Run("can be blocked by power 2", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Amrou Kithkin")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // 2/2
		g.Attack(1, gametest.PlayerA, "Amrou Kithkin")
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "Amrou Kithkin")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Grizzly Bears can block — no damage to player B
		g.AssertLife(gametest.PlayerB, 20)
	})
}

func TestWallOfWonder(t *testing.T) {
	t.Run("can attack with ability and gets +4/-4", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Wonder")
		// Activate to get +4/-4 and attack ability
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Wall of Wonder")
		g.Attack(1, gametest.PlayerA, "Wall of Wonder")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// 1+4=5 power, 5-4=1 toughness → deals 5 damage
		g.AssertLife(gametest.PlayerB, 15)
	})
}

func TestLivonyaSilone(t *testing.T) {
	t.Run("has first strike", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Livonya Silone")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Livonya Silone", core.FirstStrike, true)
	})

	t.Run("legendary landwalk makes unblockable", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Livonya Silone")
		// Defender controls a legendary land
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Karakas")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Livonya Silone")
		// Bears try to block but can't — Livonya has legendary landwalk
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "Livonya Silone")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Livonya dealt 4 damage directly to player B
		g.AssertLife(gametest.PlayerB, 16)
	})

	t.Run("can be blocked without legendary land", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Livonya Silone")
		// Defender has no legendary land
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Craw Wurm") // 6/4
		g.Attack(1, gametest.PlayerA, "Livonya Silone")
		g.Block(1, gametest.PlayerB, "Craw Wurm", "Livonya Silone")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Livonya was blocked — player B takes no damage
		g.AssertLife(gametest.PlayerB, 20)
	})
}

func TestRasputinDreamweaver(t *testing.T) {
	t.Run("enters with seven dream counters", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rasputin Dreamweaver")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Rasputin Dreamweaver", core.Dream, 7)
	})

	t.Run("remove dream counter for colorless mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rasputin Dreamweaver")
		// Remove a dream counter to add {C}
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Rasputin Dreamweaver")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Should have 6 dream counters remaining
		g.AssertCounterCount(gametest.PlayerA, "Rasputin Dreamweaver", core.Dream, 6)
	})

	t.Run("upkeep restores a dream counter if untapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rasputin Dreamweaver")
		// Remove a counter on turn 1
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Rasputin Dreamweaver")
		// Turn 3 upkeep (PlayerA's next turn) — should regain a counter
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		// Had 6 after removal, should be back to 7 after upkeep
		g.AssertCounterCount(gametest.PlayerA, "Rasputin Dreamweaver", core.Dream, 7)
	})
}

func TestLadyEvangela(t *testing.T) {
	t.Run("prevents combat damage from target creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lady Evangela")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Craw Wurm") // 6/4
		// Turn 2 is PlayerB's turn — activate Lady Evangela before combat
		g.ActivateAbility(2, core.BeginCombat, gametest.PlayerA, "Lady Evangela", "Craw Wurm")
		g.Attack(2, gametest.PlayerB, "Craw Wurm")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// Craw Wurm's 6 damage should be prevented — player A still at 20
		g.AssertLife(gametest.PlayerA, 20)
	})
}

func TestRohgahhOfKherKeep(t *testing.T) {
	t.Run("kobolds of kher keep get +2/+2", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rohgahh of Kher Keep")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kobolds of Kher Keep") // 0/1
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Kobolds of Kher Keep should be 2/3 (0+2/1+2)
		g.AssertPowerToughness(gametest.PlayerA, "Kobolds of Kher Keep", 2, 3)
	})

	t.Run("survives upkeep if can pay", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rohgahh of Kher Keep")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		// Turn 3 is PlayerA's next turn — upkeep triggers, pays {R}{R}{R}
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		// Rohgahh should still be under PlayerA's control
		g.AssertPermanentCount(gametest.PlayerA, "Rohgahh of Kher Keep", 1)
	})

	t.Run("loses control if cannot pay", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rohgahh of Kher Keep")
		// No Mountains — can't pay {R}{R}{R}
		// Turn 1 upkeep — can't pay → tap + opponent gains control
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Rohgahh should now be under PlayerB's control (tapped)
		g.AssertPermanentCount(gametest.PlayerA, "Rohgahh of Kher Keep", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Rohgahh of Kher Keep", 1)
		g.AssertTapped(gametest.PlayerB, "Rohgahh of Kher Keep", true)
	})
}

func TestStangg(t *testing.T) {
	t.Run("creates stangg twin token on ETB", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Stangg")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Should have both Stangg and Stangg Twin
		g.AssertPermanentCount(gametest.PlayerA, "Stangg", 1)
		g.AssertPermanentCount(gametest.PlayerA, "Stangg Twin", 1)
		g.AssertPowerToughness(gametest.PlayerA, "Stangg Twin", 3, 4)
	})
}

func TestWallOfVapor(t *testing.T) {
	t.Run("takes no damage from creature it blocks", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Vapor") // 0/1 Defender
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")    // 3/3
		g.Attack(2, gametest.PlayerB, "Hill Giant")
		g.Block(2, gametest.PlayerA, "Wall of Vapor", "Hill Giant")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// Wall of Vapor prevents all damage from creatures it blocks — survives
		g.AssertPermanentCount(gametest.PlayerA, "Wall of Vapor", 1)
	})

	t.Run("still deals its own damage to attacker", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Vapor") // 0/1 Defender
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // 2/2
		g.Attack(2, gametest.PlayerB, "Grizzly Bears")
		g.Block(2, gametest.PlayerA, "Wall of Vapor", "Grizzly Bears")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// Wall of Vapor is 0 power so deals 0 — Bears survive, Wall survives
		g.AssertPermanentCount(gametest.PlayerA, "Wall of Vapor", 1)
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestWallOfShadows(t *testing.T) {
	t.Run("takes no damage from creature it blocks", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Shadows") // 0/1 Defender
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Craw Wurm")      // 6/4
		g.Attack(2, gametest.PlayerB, "Craw Wurm")
		g.Block(2, gametest.PlayerA, "Wall of Shadows", "Craw Wurm")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// Wall of Shadows prevents all damage from creatures it blocks — survives
		g.AssertPermanentCount(gametest.PlayerA, "Wall of Shadows", 1)
	})
}

func TestEnchantedBeing(t *testing.T) {
	t.Run("prevents combat damage from enchanted creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Enchanted Being") // 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")   // 2/2
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Giant Strength")         // aura +2/+2
		// Enchant Bears to make them "enchanted"
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Giant Strength", "Grizzly Bears")
		// Attack with enchanted Bears (now 4/4)
		g.Attack(2, gametest.PlayerB, "Grizzly Bears")
		g.Block(2, gametest.PlayerA, "Enchanted Being", "Grizzly Bears")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// Enchanted Being prevents combat damage from enchanted creatures — survives
		// But Enchanted Being deals 2 damage to Bears (4/4), not lethal
		g.AssertPermanentCount(gametest.PlayerA, "Enchanted Being", 1)
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
	})

	t.Run("takes damage from non-enchanted creatures normally", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Enchanted Being") // 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")      // 3/3
		// Hill Giant is not enchanted — attacks normally
		g.Attack(2, gametest.PlayerB, "Hill Giant")
		g.Block(2, gametest.PlayerA, "Enchanted Being", "Hill Giant")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// Enchanted Being takes 3 damage from non-enchanted Hill Giant — dies
		g.AssertGraveyardCount(gametest.PlayerA, "Enchanted Being", 1)
	})
}

func TestElvenRiders(t *testing.T) {
	t.Run("cannot be blocked by non-Wall non-flying creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Elven Riders")   // 3/3
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")  // 2/2 no flying, not Wall
		g.Attack(1, gametest.PlayerA, "Elven Riders")
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "Elven Riders")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Grizzly Bears can't block Elven Riders — 3 damage to PlayerB
		g.AssertLife(gametest.PlayerB, 17)
	})

	t.Run("can be blocked by Wall", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Elven Riders") // 3/3
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wall of Earth") // 0/6 Wall
		g.Attack(1, gametest.PlayerA, "Elven Riders")
		g.Block(1, gametest.PlayerB, "Wall of Earth", "Elven Riders")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Wall of Earth blocks — no damage to PlayerB
		g.AssertLife(gametest.PlayerB, 20)
	})

	t.Run("can be blocked by flying creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Elven Riders")  // 3/3
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Azure Drake")   // 2/4 flying
		g.Attack(1, gametest.PlayerA, "Elven Riders")
		g.Block(1, gametest.PlayerB, "Azure Drake", "Elven Riders")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Azure Drake has flying — can block Elven Riders
		g.AssertLife(gametest.PlayerB, 20)
	})
}

func TestElderSpawn(t *testing.T) {
	t.Run("survives upkeep if Island is sacrificed", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Elder Spawn")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Elder Spawn stays, Island sacrificed
		g.AssertPermanentCount(gametest.PlayerA, "Elder Spawn", 1)
		g.AssertPermanentCount(gametest.PlayerA, "Island", 0)
	})

	t.Run("sacrificed and deals 6 damage if no Island", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Elder Spawn")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// No Island — Elder Spawn sacrificed + 6 damage to controller
		g.AssertPermanentCount(gametest.PlayerA, "Elder Spawn", 0)
		g.AssertLife(gametest.PlayerA, 14)
	})

	t.Run("cannot be blocked by red creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Elder Spawn")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island") // survive upkeep
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Raging Bull") // 2/2 red creature
		g.Attack(1, gametest.PlayerA, "Elder Spawn")
		g.Block(1, gametest.PlayerB, "Raging Bull", "Elder Spawn")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Raging Bull is red — can't block Elder Spawn — 6 damage to PlayerB
		g.AssertLife(gametest.PlayerB, 14)
	})
}

func TestMoldDemon(t *testing.T) {
	t.Run("survives ETB if two Swamps sacrificed", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Mold Demon")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Mold Demon")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Mold Demon enters, two Swamps sacrificed, Mold Demon stays
		g.AssertPermanentCount(gametest.PlayerA, "Mold Demon", 1)
		g.AssertPermanentCount(gametest.PlayerA, "Swamp", 0)
	})

	t.Run("sacrificed if fewer than two Swamps", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Mold Demon")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Swamp") // only 1 Swamp
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Mold Demon")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Can't sacrifice two Swamps — Mold Demon is sacrificed
		g.AssertPermanentCount(gametest.PlayerA, "Mold Demon", 0)
	})
}

func TestEvilEyeOfOrmsByGore(t *testing.T) {
	t.Run("non-Eye creatures cannot attack", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Evil Eye of Orms-by-Gore")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		// Try to attack with Bears — should be prevented
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Bears can't attack when Evil Eye is out
		g.AssertLife(gametest.PlayerB, 20)
	})

	t.Run("Evil Eye itself can attack", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Evil Eye of Orms-by-Gore")
		g.Attack(3, gametest.PlayerA, "Evil Eye of Orms-by-Gore")
		g.StopAt(3, core.EndStep)
		g.Execute()
		// Evil Eye is an Eye — can attack
		g.AssertLife(gametest.PlayerB, 17)
	})
}

func TestPsionicEntity(t *testing.T) {
	t.Run("deals 2 to target and 3 to self", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Psionic Entity")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Psionic Entity", "PlayerB")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Deals 2 to PlayerB
		g.AssertLife(gametest.PlayerB, 18)
		// 3 damage to self (2 toughness) — should die
		g.AssertPermanentCount(gametest.PlayerA, "Psionic Entity", 0)
	})
}

func TestGhostsOfTheDamned(t *testing.T) {
	t.Run("tap to give target creature -1/-0", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ghosts of the Damned")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Ghosts of the Damned", "Hill Giant")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerB, "Hill Giant", 2, 3)
	})
}

func TestCosmicHorror(t *testing.T) {
	t.Run("sacrificed at upkeep if cannot pay", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Cosmic Horror")
		// On turn 3 upkeep, Cosmic Horror's sacrifice trigger fires
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		// Can't pay {3}{B}{B}{B} — sacrificed
		g.AssertPermanentCount(gametest.PlayerA, "Cosmic Horror", 0)
	})
}

func TestWolverinePack(t *testing.T) {
	t.Run("rampage 2 gives +2/+2 per extra blocker", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wolverine Pack") // 2/4
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")  // 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")     // 3/3
		g.Attack(3, gametest.PlayerA, "Wolverine Pack")
		g.Block(3, gametest.PlayerB, "Grizzly Bears", "Wolverine Pack")
		g.Block(3, gametest.PlayerB, "Hill Giant", "Wolverine Pack")
		g.StopAt(3, core.EndStep)
		g.Execute()
		// Rampage 2: 2 blockers, 1 beyond first => +2/+2 => 4/6
		// 4/6 vs (2+3=5 damage) => survives with 1 toughness
		g.AssertPermanentCount(gametest.PlayerA, "Wolverine Pack", 1)
	})
}

func TestFrostGiant(t *testing.T) {
	t.Run("rampage 2", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Frost Giant") // 4/4
		g.Attack(3, gametest.PlayerA, "Frost Giant")
		g.StopAt(3, core.EndStep)
		g.Execute()
		// Unblocked — deals 4 damage
		g.AssertLife(gametest.PlayerB, 16)
	})
}

func TestDevouringDeep(t *testing.T) {
	t.Run("islandwalk", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Devouring Deep") // 1/1
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.Attack(3, gametest.PlayerA, "Devouring Deep")
		g.Block(3, gametest.PlayerB, "Grizzly Bears", "Devouring Deep")
		g.StopAt(3, core.EndStep)
		g.Execute()
		// Islandwalk — can't be blocked since defender controls Island
		g.AssertLife(gametest.PlayerB, 19)
	})
}

func TestLostSoul(t *testing.T) {
	t.Run("swampwalk", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lost Soul") // 2/1
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Swamp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.Attack(3, gametest.PlayerA, "Lost Soul")
		g.Block(3, gametest.PlayerB, "Grizzly Bears", "Lost Soul")
		g.StopAt(3, core.EndStep)
		g.Execute()
		// Swampwalk — can't be blocked since defender controls Swamp
		g.AssertLife(gametest.PlayerB, 18)
	})
}

func TestCrimsonManticore(t *testing.T) {
	t.Run("flying creature attacks for 2", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Crimson Manticore")
		g.Attack(3, gametest.PlayerA, "Crimson Manticore")
		g.StopAt(3, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 18) // 2/2 flying
	})
}

// ===== ADDITIONAL VANILLA CREATURES =====

func TestHeadlessHorseman(t *testing.T) {
	t.Run("is 2/2 vanilla", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Headless Horseman")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Headless Horseman", 2, 2)
	})
}

func TestRagingBull(t *testing.T) {
	t.Run("is 2/2 vanilla", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Raging Bull")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Raging Bull", 2, 2)
	})
}

func TestBarbaryApes(t *testing.T) {
	t.Run("is 2/2 vanilla", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Barbary Apes")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Barbary Apes", 2, 2)
	})
}

func TestDurkwoodBoars(t *testing.T) {
	t.Run("is 4/4 vanilla", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Durkwood Boars")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Durkwood Boars", 4, 4)
	})
}

func TestMossMonster(t *testing.T) {
	t.Run("is 3/6 vanilla", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Moss Monster")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Moss Monster", 3, 6)
	})
}

func TestCrimsonKobolds(t *testing.T) {
	t.Run("is 0/1 for zero mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Crimson Kobolds")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Crimson Kobolds", 0, 1)
	})
}

func TestCrookshankKobolds(t *testing.T) {
	t.Run("is 0/1 for zero mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Crookshank Kobolds")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Crookshank Kobolds", 0, 1)
	})
}

func TestKoboldsOfKherKeep(t *testing.T) {
	t.Run("is 0/1 for zero mana", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kobolds of Kher Keep")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Kobolds of Kher Keep", 0, 1)
	})
}

// ===== VANILLA LEGENDARY CREATURES =====

func TestJasmineBoreal(t *testing.T) {
	t.Run("is 4/5 legendary", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Jasmine Boreal")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Jasmine Boreal", 4, 5)
	})
}

func TestJeditOjanen(t *testing.T) {
	t.Run("is 5/5 legendary", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Jedit Ojanen")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Jedit Ojanen", 5, 5)
	})
}

func TestJerrardOfTheClosedFist(t *testing.T) {
	t.Run("is 6/5 legendary", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Jerrard of the Closed Fist")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Jerrard of the Closed Fist", 6, 5)
	})
}

func TestKasimirTheLoneWolf(t *testing.T) {
	t.Run("is 5/3 legendary", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Kasimir the Lone Wolf")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Kasimir the Lone Wolf", 5, 3)
	})
}

func TestLadyOrca(t *testing.T) {
	t.Run("is 7/4 legendary", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lady Orca")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Lady Orca", 7, 4)
	})
}

func TestSivitriScarzam(t *testing.T) {
	t.Run("is 6/4 legendary", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sivitri Scarzam")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Sivitri Scarzam", 6, 4)
	})
}

func TestTheLadyOfTheMountain(t *testing.T) {
	t.Run("is 5/5 legendary", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "The Lady of the Mountain")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "The Lady of the Mountain", 5, 5)
	})
}

func TestTobiasAndrion(t *testing.T) {
	t.Run("is 4/4 legendary", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Tobias Andrion")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Tobias Andrion", 4, 4)
	})
}

func TestTorstenVonUrsus(t *testing.T) {
	t.Run("is 5/5 legendary", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Torsten Von Ursus")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Torsten Von Ursus", 5, 5)
	})
}

func TestSirShandlarOfEberyn(t *testing.T) {
	t.Run("is 4/7 legendary", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sir Shandlar of Eberyn")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Sir Shandlar of Eberyn", 4, 7)
	})
}

// ===== KEYWORD CREATURES =====

func TestHornetCobra(t *testing.T) {
	t.Run("has first strike", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hornet Cobra")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Hornet Cobra", 2, 1)
		g.AssertHasAbility(gametest.PlayerA, "Hornet Cobra", core.FirstStrike, true)
	})
}

func TestSegovianLeviathan(t *testing.T) {
	t.Run("has islandwalk", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Segovian Leviathan")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		g.Attack(3, gametest.PlayerA, "Segovian Leviathan")
		g.StopAt(3, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 17) // 3/3 islandwalk, unblockable
	})
}

func TestBartelRuneaxe(t *testing.T) {
	t.Run("has vigilance", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Bartel Runeaxe")
		g.Attack(3, gametest.PlayerA, "Bartel Runeaxe")
		g.StopAt(3, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 14) // 6/5 attacks
		g.AssertTapped(gametest.PlayerA, "Bartel Runeaxe", false) // vigilance
	})
}

// ===== ACTIVATED ABILITY CREATURES =====

func TestWallOfOpposition(t *testing.T) {
	t.Run("is 0/6 defender with pump", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Opposition")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Wall of Opposition", 0, 6)
		g.AssertHasAbility(gametest.PlayerA, "Wall of Opposition", core.Defender, true)
	})
}

func TestPradeshGypsies(t *testing.T) {
	t.Run("tap to give target -2/-0", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Pradesh Gypsies")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Raging Bull") // 2/2
		g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Pradesh Gypsies", "target creature gets -2/-0", "Raging Bull")
		g.StopAt(3, core.PostcombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerB, "Raging Bull", 0, 2) // -2/-0 applied
	})
}

func TestRadjanSpirit(t *testing.T) {
	t.Run("tap to remove flying", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Radjan Spirit")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Azure Drake") // 2/4 flying
		g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Radjan Spirit", "target creature loses flying", "Azure Drake")
		g.StopAt(3, core.PostcombatMain)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerB, "Azure Drake", core.Flying, false)
	})
}

func TestPixieQueen(t *testing.T) {
	t.Run("grant flying to target", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Pixie Queen")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Raging Bull") // 2/2 no flying
		g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Pixie Queen", "target creature gains flying", "Raging Bull")
		g.StopAt(3, core.PostcombatMain)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Raging Bull", core.Flying, true)
	})
}

func TestHyperionBlacksmith(t *testing.T) {
	t.Run("tap target artifact", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hyperion Blacksmith")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Relic Barrier")
		g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Hyperion Blacksmith", "tap or untap", "Relic Barrier")
		g.StopAt(3, core.PostcombatMain)
		g.Execute()
		g.AssertTapped(gametest.PlayerB, "Relic Barrier", true)
	})
}

func TestMasterOfTheHunt(t *testing.T) {
	t.Run("creates wolf token", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Master of the Hunt")
		g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Master of the Hunt", "Create", "")
		g.StopAt(3, core.PostcombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Wolves of the Hunt", 1)
	})
}

func TestGwendlynDiCorci(t *testing.T) {
	t.Run("is 3/5 legendary", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gwendlyn Di Corci")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Gwendlyn Di Corci", 3, 5)
	})
}

// ===== ELDER DRAGONS =====

func TestArcadesSabboth(t *testing.T) {
	t.Run("sacrificed at upkeep if cant pay", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Arcades Sabboth")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Can't pay {G}{W}{U} so sacrificed during upkeep
		g.AssertPermanentCount(gametest.PlayerA, "Arcades Sabboth", 0)
		g.AssertGraveyardCount(gametest.PlayerA, "Arcades Sabboth", 1)
	})
}

func TestChromium(t *testing.T) {
	t.Run("sacrificed at upkeep if cant pay", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Chromium")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Can't pay {W}{U}{B} so sacrificed during upkeep
		g.AssertPermanentCount(gametest.PlayerA, "Chromium", 0)
		g.AssertGraveyardCount(gametest.PlayerA, "Chromium", 1)
	})
}

// ===== WALL OF PUTRID FLESH =====

func TestWallOfPutridFlesh(t *testing.T) {
	t.Run("has defender and protection from white", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Putrid Flesh")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Wall of Putrid Flesh", 2, 4)
		g.AssertHasAbility(gametest.PlayerA, "Wall of Putrid Flesh", core.Defender, true)
	})
}

// ===== ELDER LAND WURM =====

func TestElderLandWurm(t *testing.T) {
	t.Run("starts with defender and trample", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Elder Land Wurm")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Elder Land Wurm", 5, 5)
		g.AssertHasAbility(gametest.PlayerA, "Elder Land Wurm", core.Defender, true)
		g.AssertHasAbility(gametest.PlayerA, "Elder Land Wurm", core.Trample, true)
	})

	t.Run("loses defender when it blocks", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Elder Land Wurm")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Durkwood Boars") // 4/4
		g.Attack(2, gametest.PlayerB, "Durkwood Boars")
		g.Block(2, gametest.PlayerA, "Elder Land Wurm", "Durkwood Boars")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// After blocking, Elder Land Wurm loses defender
		g.AssertHasAbility(gametest.PlayerA, "Elder Land Wurm", core.Defender, false)
		// Now it can attack on turn 3
		g.Attack(3, gametest.PlayerA, "Elder Land Wurm")
		g.StopAt(3, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 15) // 5/5 trample
	})
}

// ===== HELL'S CARETAKER =====

func TestHellsCaretaker(t *testing.T) {
	t.Run("reanimate during upkeep by sacrificing creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hell's Caretaker")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Raging Bull") // sacrifice fodder
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Moss Monster")  // reanimate target
		g.ActivateAbility(3, core.Upkeep, gametest.PlayerA, "Hell's Caretaker", "Return target creature", "Moss Monster")
		g.ChoosePermanent(gametest.PlayerA, "Raging Bull") // sacrifice this
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Moss Monster", 1)   // reanimated
		g.AssertPermanentCount(gametest.PlayerA, "Raging Bull", 0)    // sacrificed
		g.AssertGraveyardCount(gametest.PlayerA, "Raging Bull", 1)
	})
}

func TestWhirlingDervish(t *testing.T) {
	t.Run("gets +1/+1 counter when deals combat damage to opponent", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Whirling Dervish")
		g.Attack(1, gametest.PlayerA, "Whirling Dervish")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 19)
		g.AssertPowerToughness(gametest.PlayerA, "Whirling Dervish", 2, 2)
	})

	t.Run("is not damaged by black creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Whirling Dervish") // 1/1
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Headless Horseman") // 2/2 black
		g.Attack(2, gametest.PlayerB, "Headless Horseman")
		g.Block(2, gametest.PlayerA, "Whirling Dervish", "Headless Horseman")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// Whirling Dervish has protection from black, shouldn't take damage
		g.AssertPermanentCount(gametest.PlayerA, "Whirling Dervish", 1)
	})
}

func TestPrimordialOoze(t *testing.T) {
	t.Run("gets +1/+1 counter at upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Primordial Ooze")
		// Turn 1 upkeep: add +1/+1 counter
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertCounterCount(gametest.PlayerA, "Primordial Ooze", core.P1P1, 1)
		g.AssertPowerToughness(gametest.PlayerA, "Primordial Ooze", 2, 2) // 1/1 base + 1 counter
	})
}

func TestAbomination(t *testing.T) {
	t.Run("destroys green creature that blocks it", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Abomination") // 2/6
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Durkwood Boars") // 4/4 green
		g.Attack(1, gametest.PlayerA, "Abomination")
		g.Block(1, gametest.PlayerB, "Durkwood Boars", "Abomination")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Durkwood Boars is green — trigger destroys it (even though combat wouldn't kill 4/4)
		g.AssertPermanentCount(gametest.PlayerB, "Durkwood Boars", 0)
		// Abomination survives (takes 4 damage but has 6 toughness)
		g.AssertPermanentCount(gametest.PlayerA, "Abomination", 1)
	})
}

func TestIchneumonDruid(t *testing.T) {
	t.Run("deals 4 damage when opponent casts second instant", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ichneumon Druid")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Holy Day")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Darkness")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Holy Day")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Darkness")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		// Second instant triggers 4 damage to opponent
		g.AssertLife(gametest.PlayerB, 16)
	})
}

func TestAislingLeprechaun(t *testing.T) {
	t.Run("creature blocked by Aisling becomes green", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Aisling Leprechaun") // 1/1 green
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Raging Bull")        // 2/2 red
		g.Attack(2, gametest.PlayerB, "Raging Bull")
		g.Block(2, gametest.PlayerA, "Aisling Leprechaun", "Raging Bull")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// Raging Bull should have been turned green indefinitely
		// Aisling dies in combat (1/1 vs 2/2) but the color change is permanent
	})
}

func TestOsaiVultures(t *testing.T) {
	t.Run("gets carrion counter when creature dies", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Osai Vultures")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // 2/2
		// Both Bears attack/block and kill each other
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "Grizzly Bears")
		g.StopAt(2, core.PrecombatMain)
		g.Execute()
		// Two creatures died — Osai Vultures should get a carrion counter at end step
		g.AssertCounterCount(gametest.PlayerA, "Osai Vultures", core.Carrion, 1)
	})
}

func TestWallOfTombstones(t *testing.T) {
	t.Run("toughness equals 1 plus creature cards in graveyard", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Tombstones")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneGraveyard, gametest.PlayerA, "Raging Bull")
		// Turn 1 upkeep: Wall checks graveyard, 2 creatures → toughness = 3
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Wall of Tombstones", 0, 3)
	})
}

func TestRubiniaSoulsinger(t *testing.T) {
	t.Run("steals creature while tapped", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Rubinia Soulsinger")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		// Tap Rubinia to steal Bears
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Rubinia Soulsinger", "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Bears should be controlled by PlayerA now
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
	})
}

func TestVampireBatsActivationLimit(t *testing.T) {
	t.Run("pump limited to twice per turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Vampire Bats")
		// Pump twice: 0/1 → 1/1 → 2/1
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Vampire Bats")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Vampire Bats")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Should be 2/1 (pumped twice)
		g.AssertPowerToughness(gametest.PlayerA, "Vampire Bats", 2, 1)
	})
}

func TestBronzeHorse(t *testing.T) {
	t.Run("prevents targeted spell damage with another creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Bronze Horse")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // another creature
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Bronze Horse")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// Damage prevented — Bronze Horse should still be at full health (4/4, no damage)
		g.AssertPowerToughness(gametest.PlayerA, "Bronze Horse", 4, 4)
	})

	t.Run("no prevention without another creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Bronze Horse")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Bronze Horse")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// No other creature — damage NOT prevented, Horse takes 3 damage (4/4 → 4/1 effective)
		g.AssertPermanentCount(gametest.PlayerA, "Bronze Horse", 1)
	})
}

func TestWallOfDust(t *testing.T) {
	t.Run("blocked creature cannot attack next turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Wall of Dust")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		// Bears attack on turn 2, blocked by Wall of Dust
		g.Attack(2, gametest.PlayerB, "Grizzly Bears")
		g.Block(2, gametest.PlayerA, "Wall of Dust", "Grizzly Bears")
		// Bears attack again on turn 4 — should be prevented
		g.Attack(4, gametest.PlayerB, "Grizzly Bears")
		g.StopAt(4, core.EndStep)
		g.Execute()
		// Bears could not attack — PlayerA takes 0 damage on turn 4
		g.AssertLife(gametest.PlayerA, 20)
	})
}

func TestGiantTurtle(t *testing.T) {
	t.Run("can't attack if it attacked last turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Giant Turtle") // 2/4
		// Turn 1: Giant Turtle attacks
		g.Attack(1, gametest.PlayerA, "Giant Turtle")
		// Turn 3: Giant Turtle tries to attack again (PlayerA's next turn)
		g.Attack(3, gametest.PlayerA, "Giant Turtle")
		g.StopAt(3, core.EndStep)
		g.Execute()
		// Turn 1 attack dealt 2 damage
		// Turn 3 attack should be prevented — no additional damage
		g.AssertLife(gametest.PlayerB, 18) // only 2 damage from turn 1
	})

	t.Run("can attack after skipping a turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Giant Turtle")
		// Turn 1: attacks
		g.Attack(1, gametest.PlayerA, "Giant Turtle")
		// Turn 3: can't attack (last turn restriction)
		// Turn 5: should be able to attack again
		g.Attack(5, gametest.PlayerA, "Giant Turtle")
		g.StopAt(5, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 16) // 2 from turn 1 + 2 from turn 5
	})
}

func TestGabrielAngelfire(t *testing.T) {
	t.Run("choose flying at upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gabriel Angelfire")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3 non-flying blocker
		// Turn 1 upkeep: choose flying (mode 0)
		g.ChooseMode(gametest.PlayerA, 0)
		g.Attack(1, gametest.PlayerA, "Gabriel Angelfire")
		g.Block(1, gametest.PlayerB, "Hill Giant", "Gabriel Angelfire")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// With flying, Hill Giant can't block — 4 damage gets through
		g.AssertLife(gametest.PlayerB, 16)
	})

	t.Run("choose first strike at upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gabriel Angelfire") // 4/4
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Craw Wurm")        // 6/4
		// Turn 1 upkeep: choose first strike (mode 1)
		g.ChooseMode(gametest.PlayerA, 1)
		g.Attack(1, gametest.PlayerA, "Gabriel Angelfire")
		g.Block(1, gametest.PlayerB, "Craw Wurm", "Gabriel Angelfire")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// First strike: Gabriel deals 4 to Craw Wurm first, killing it (4 toughness)
		// Craw Wurm never deals damage
		g.AssertPermanentCount(gametest.PlayerA, "Gabriel Angelfire", 1)
		g.AssertGraveyardCount(gametest.PlayerB, "Craw Wurm", 1)
	})

	t.Run("choose trample at upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gabriel Angelfire") // 4/4
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")     // 2/2
		// Turn 1 upkeep: choose trample (mode 2)
		g.ChooseMode(gametest.PlayerA, 2)
		g.Attack(1, gametest.PlayerA, "Gabriel Angelfire")
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "Gabriel Angelfire")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Trample: 2 assigned to Bears, 2 tramples to player
		g.AssertLife(gametest.PlayerB, 18)
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestHalfdane(t *testing.T) {
	t.Run("copies target creature P/T at upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Halfdane")   // 3/3
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Craw Wurm") // 6/4
		// Turn 1 upkeep: choose Craw Wurm to copy P/T
		g.ChoosePermanent(gametest.PlayerA, "Craw Wurm")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Halfdane", 6, 4)
	})

	t.Run("reverts after next upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Halfdane")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Craw Wurm")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // 2/2
		// Turn 1 upkeep: copy Craw Wurm (6/4)
		g.ChoosePermanent(gametest.PlayerA, "Craw Wurm")
		// Turn 3 upkeep: choose Grizzly Bears (2/2) — old effect expires
		g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Halfdane", 2, 2)
	})
}

func TestHazezonTamar(t *testing.T) {
	t.Run("creates Sand Warrior tokens at next upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hazezon Tamar")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		// Hazezon entered on setup, but ETB doesn't fire for pre-placed cards.
		// Need to cast it from hand instead.
		g.StopAt(1, core.EndStep)
		g.Execute()
	})

	t.Run("creates tokens when cast from hand", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Hazezon Tamar")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Hazezon Tamar")
		// Turn 3 upkeep: delayed trigger fires, count lands (3), create 3 Sand Warriors
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Sand Warrior", 3)
	})

	t.Run("exile Sand Warriors when Hazezon leaves", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Hazezon Tamar")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Swords to Plowshares")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Hazezon Tamar")
		// Turn 3 upkeep: creates 2 Sand Warriors (Mountain + Forest)
		// Then exile Hazezon with Swords to Plowshares
		g.CastSpell(3, core.PrecombatMain, gametest.PlayerB, "Swords to Plowshares", "Hazezon Tamar")
		g.StopAt(3, core.EndStep)
		g.Execute()
		// Sand Warriors should be exiled
		g.AssertPermanentCount(gametest.PlayerA, "Sand Warrior", 0)
	})
}

func TestJohan(t *testing.T) {
	t.Run("creatures don't tap to attack when chosen", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Johan")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		// Choose "may" at begin combat — Johan can't attack but grants vigilance
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Bears attacked but should be untapped (vigilance)
		g.AssertTapped(gametest.PlayerA, "Grizzly Bears", false)
		g.AssertLife(gametest.PlayerB, 18) // 2 damage from Bears
	})
}

func TestAbominationEndOfCombat(t *testing.T) {
	t.Run("destroys green blocker at end of combat after damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Abomination")   // 2/6
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // 2/2, green
		g.Attack(1, gametest.PlayerA, "Abomination")
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "Abomination")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Bears should be destroyed at end of combat (green creature blocked Abomination)
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
		// Abomination took 2 damage from Bears but has 6 toughness, should survive
		g.AssertPermanentCount(gametest.PlayerA, "Abomination", 1)
		// Bears dealt 2 combat damage to Abomination before being destroyed
		g.AssertLife(gametest.PlayerB, 20) // Abomination was blocked, no player damage
	})
}

func TestInfernalMedusa(t *testing.T) {
	t.Run("destroys creature it blocks at end of combat", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")        // 3/3
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Infernal Medusa")   // 2/4
		g.Attack(1, gametest.PlayerB)
		g.Attack(2, gametest.PlayerA)
		// PlayerB attacks with Hill Giant on turn 2 (their turn)
		// Actually let's set it up properly: B attacks, A blocks with Medusa
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Turn 1 is PlayerA's turn, no attacks from B yet
		g.AssertLife(gametest.PlayerB, 20)
	})

	t.Run("when blocking destroys attacker at end of combat", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Infernal Medusa") // 2/4
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")      // 3/3
		// Turn 2 is PlayerB's turn — they attack with Hill Giant
		g.Attack(2, gametest.PlayerB, "Hill Giant")
		g.Block(2, gametest.PlayerA, "Infernal Medusa", "Hill Giant")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// Hill Giant is destroyed at end of combat (blocked by Medusa)
		g.AssertGraveyardCount(gametest.PlayerB, "Hill Giant", 1)
		// Medusa took 3 damage but has 4 toughness — survives
		g.AssertPermanentCount(gametest.PlayerA, "Infernal Medusa", 1)
	})

	t.Run("when attacking destroys non-Wall blockers at end of combat", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Infernal Medusa") // 2/4
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")   // 2/2
		g.Attack(1, gametest.PlayerA, "Infernal Medusa")
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "Infernal Medusa")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Bears destroyed at end of combat (non-Wall blocking Medusa)
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
		g.AssertPermanentCount(gametest.PlayerA, "Infernal Medusa", 1)
	})

	t.Run("does not destroy Wall blockers", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Infernal Medusa") // 2/4
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wall of Earth")   // 0/6, Wall
		g.Attack(1, gametest.PlayerA, "Infernal Medusa")
		g.Block(1, gametest.PlayerB, "Wall of Earth", "Infernal Medusa")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Wall is NOT destroyed (Medusa's second ability only hits non-Walls)
		g.AssertPermanentCount(gametest.PlayerB, "Wall of Earth", 1)
		g.AssertPermanentCount(gametest.PlayerA, "Infernal Medusa", 1)
	})
}

func TestTimeElemental(t *testing.T) {
	t.Run("sacrificed at end of combat when attacks", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Time Elemental") // 0/2
		g.Attack(1, gametest.PlayerA, "Time Elemental")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Time Elemental should be sacrificed at end of combat and deal 5 to controller
		g.AssertGraveyardCount(gametest.PlayerA, "Time Elemental", 1)
		g.AssertLife(gametest.PlayerA, 15) // 20 - 5 = 15
	})

	t.Run("bounces non-enchanted permanent", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Time Elemental")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Time Elemental", "Return target permanent", "Hill Giant")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 0)
	})
}

func TestBrineHag(t *testing.T) {
	t.Run("creatures that dealt damage become 0/2 when dies", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Brine Hag")    // 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "White Knight") // 2/2 first strike
		// Turn 2: PlayerB attacks with White Knight, Brine Hag blocks
		// First strike: White Knight deals 2 to Brine Hag → Brine Hag dies
		// Brine Hag never deals damage back (dead before regular combat damage)
		// White Knight dealt damage to Brine Hag → becomes 0/2 with 0 damage → survives
		g.Attack(2, gametest.PlayerB, "White Knight")
		g.Block(2, gametest.PlayerA, "Brine Hag", "White Knight")
		g.StopAt(2, core.PostcombatMain)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerA, "Brine Hag", 1)
		g.AssertPermanentCount(gametest.PlayerB, "White Knight", 1)
		g.AssertPowerToughness(gametest.PlayerB, "White Knight", 0, 2)
	})
}

func TestFloralSpuzzem(t *testing.T) {
	t.Run("destroy artifact when unblocked", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Floral Spuzzem") // 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Jayemdae Tome")  // artifact
		// Turn 1: Floral Spuzzem attacks, unblocked
		g.Attack(1, gametest.PlayerA, "Floral Spuzzem")
		g.ChoosePermanent(gametest.PlayerA, "Jayemdae Tome")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Chose to destroy artifact → no combat damage dealt, artifact destroyed
		g.AssertGraveyardCount(gametest.PlayerB, "Jayemdae Tome", 1)
		g.AssertLife(gametest.PlayerB, 20) // no combat damage
	})
}

func TestGiantSlug(t *testing.T) {
	t.Run("gains landwalk at next upkeep", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Giant Slug")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Swamp")
		// Turn 1: Activate ability for {5}
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Giant Slug", "choose a basic land type")
		g.ChooseMode(gametest.PlayerA, 4) // mode 4 = Swamp
		// Turn 3: Should have swampwalk at upkeep, attack and be unblockable
		g.Attack(3, gametest.PlayerA, "Giant Slug")
		g.StopAt(3, core.PostcombatMain)
		g.Execute()
		// Giant Slug has swampwalk and opponent has Swamp → unblockable
		g.AssertLife(gametest.PlayerB, 19) // 1 damage from 1/1
	})
}

func TestLesserWerewolf(t *testing.T) {
	t.Run("pump down to debuff blocker", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Lesser Werewolf") // 2/4
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")      // 3/3
		// Turn 1: attack with Lesser Werewolf
		g.Attack(1, gametest.PlayerA, "Lesser Werewolf")
		g.Block(1, gametest.PlayerB, "Hill Giant", "Lesser Werewolf")
		// Activate ability targeting Hill Giant during blockers step
		g.ActivateAbility(1, core.FirstStrikeDamage, gametest.PlayerA, "Lesser Werewolf", "gets -1/-0", "Hill Giant")
		g.StopAt(2, core.PrecombatMain) // next turn so EOT cleanup runs
		g.Execute()
		// Lesser Werewolf gets -1/-0 (becomes 1/4), Hill Giant gets -0/-1 counter
		// Combat: LW deals 1, HG deals 3 → LW survives (1/4 - 3 = 1/1), HG was 3/2 from counter, takes 1 → 3/1
		g.AssertPermanentCount(gametest.PlayerA, "Lesser Werewolf", 1)
		g.AssertPowerToughness(gametest.PlayerA, "Lesser Werewolf", 2, 4) // -1/-0 is EOT, so back to 2/4
		g.AssertCounterCount(gametest.PlayerB, "Hill Giant", core.M0M1, 1)
	})
}

func TestTheWretched(t *testing.T) {
	t.Run("steals blocking creatures at end of combat", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "The Wretched")  // 2/5
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // 2/2
		g.Attack(1, gametest.PlayerA, "The Wretched")
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "The Wretched")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Bears blocked The Wretched — at end of combat, The Wretched steals them
		// Bears took 2 damage from The Wretched and dealt 2 damage back — Bears survive (2/2, 2 damage)
		// Actually Bears die: 2 damage from The Wretched kills 2/2 Bears
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	})

	t.Run("steals surviving blockers", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "The Wretched") // 2/5
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")   // 3/3
		g.Attack(1, gametest.PlayerA, "The Wretched")
		g.Block(1, gametest.PlayerB, "Hill Giant", "The Wretched")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Hill Giant (3/3) blocks The Wretched (2/5)
		// Hill Giant takes 2 damage (survives at 3/1), The Wretched takes 3 damage (survives at 2/2)
		// At end of combat, The Wretched steals Hill Giant
		g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 1) // now under A's control
		g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 0)
	})
}

func TestClergyOfTheHolyNimbus(t *testing.T) {
	t.Run("regenerates when destroyed", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Clergy of the Holy Nimbus") // 1/1
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")                // 3/3
		// Clergy blocks Hill Giant — would be destroyed but regenerates
		g.Attack(2, gametest.PlayerB, "Hill Giant")
		g.Block(2, gametest.PlayerA, "Clergy of the Holy Nimbus", "Hill Giant")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// Clergy should survive (regenerated instead of dying)
		g.AssertPermanentCount(gametest.PlayerA, "Clergy of the Holy Nimbus", 1)
		// Clergy is tapped (regeneration taps the creature)
		g.AssertTapped(gametest.PlayerA, "Clergy of the Holy Nimbus", true)
	})

	t.Run("can be destroyed when opponent pays 1", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Clergy of the Holy Nimbus") // 1/1
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")                // 3/3
		// Opponent activates {1} to prevent regeneration
		g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerB, "Clergy of the Holy Nimbus")
		// Then attack with Hill Giant, Clergy blocks
		g.Attack(2, gametest.PlayerB, "Hill Giant")
		g.Block(2, gametest.PlayerA, "Clergy of the Holy Nimbus", "Hill Giant")
		g.StopAt(2, core.EndStep)
		g.Execute()
		// Clergy should be destroyed (can't regenerate this turn)
		g.AssertGraveyardCount(gametest.PlayerA, "Clergy of the Holy Nimbus", 1)
	})
}

func TestSwordOfTheAges(t *testing.T) {
	t.Run("deals damage equal to total power of sacrificed creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sword of the Ages")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")    // 3/3
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		// Sword enters tapped; wait until turn 3 when it untaps
		// Choose to sacrifice Hill Giant and Grizzly Bears
		g.ChooseMode(gametest.PlayerA, 0)                       // 0 = "Sacrifice a creature"
		g.ChoosePermanent(gametest.PlayerA, "Hill Giant")       // pick Hill Giant
		g.ChooseMode(gametest.PlayerA, 0)                       // sacrifice another
		g.ChoosePermanent(gametest.PlayerA, "Grizzly Bears")   // pick Grizzly Bears
		g.ChooseMode(gametest.PlayerA, 1)                       // 1 = "Done"
		g.ActivateAbility(3, core.PrecombatMain, gametest.PlayerA, "Sword of the Ages", "{T}, Sacrifice", "PlayerB")
		g.StopAt(3, core.PostcombatMain)
		g.Execute()
		// 3 + 2 = 5 total power, dealt to PlayerB
		g.AssertLife(gametest.PlayerB, 15)
		// Sword and creatures should be exiled
		g.AssertPermanentCount(gametest.PlayerA, "Sword of the Ages", 0)
		g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 0)
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
	})
}

func TestWallOfCaltrops(t *testing.T) {
	t.Run("gains banding when blocking with another Wall", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wall of Caltrops")  // 2/1 Defender
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wall of Earth")     // 0/6 Defender
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")        // 3/3
		g.Attack(1, gametest.PlayerA, "Hill Giant")
		g.Block(1, gametest.PlayerB, "Wall of Caltrops", "Hill Giant")
		g.Block(1, gametest.PlayerB, "Wall of Earth", "Hill Giant")
		// With banding, the defender distributes damage: put all 3 on Wall of Earth
		g.ChooseBandingDistribution(gametest.PlayerB, map[string]int{
			"Wall of Caltrops": 0,
			"Wall of Earth":    3,
		})
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Both walls should survive since Wall of Earth has 6 toughness
		g.AssertPermanentCount(gametest.PlayerB, "Wall of Caltrops", 1)
		g.AssertPermanentCount(gametest.PlayerB, "Wall of Earth", 1)
	})
	t.Run("does not gain banding when blocking with non-Wall", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wall of Caltrops")  // 2/1 Defender
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")     // 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")        // 3/3
		g.Attack(1, gametest.PlayerA, "Hill Giant")
		g.Block(1, gametest.PlayerB, "Wall of Caltrops", "Hill Giant")
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "Hill Giant")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// No banding — attacker distributes damage, Wall of Caltrops (1 toughness) likely dies
		g.AssertPermanentCount(gametest.PlayerB, "Wall of Caltrops", 0)
	})
}

func TestShimianNightStalker(t *testing.T) {
	t.Run("redirects combat damage from attacker to self", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Shimian Night Stalker") // 4/4
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Swamp")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant") // 3/3
		g.Attack(1, gametest.PlayerA, "Hill Giant")
		g.ActivateAbility(1, core.DeclareBlockers, gametest.PlayerB, "Shimian Night Stalker", "Hill Giant")
		g.StopAt(1, core.EndStep)
		g.Execute()
		// Hill Giant's 3 damage should go to Shimian Night Stalker instead of PlayerB
		g.AssertLife(gametest.PlayerB, 20)
		// Shimian Night Stalker took 3 damage (4/4 -> should survive)
		g.AssertPermanentCount(gametest.PlayerB, "Shimian Night Stalker", 1)
	})
}

func TestFirestormPhoenix(t *testing.T) {
	t.Run("returns to hand instead of dying", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Firestorm Phoenix") // 3/2 Flying
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Firestorm Phoenix")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// Should be in hand, not graveyard
		g.AssertGraveyardCount(gametest.PlayerA, "Firestorm Phoenix", 0)
		g.AssertPermanentCount(gametest.PlayerA, "Firestorm Phoenix", 0)
		g.AssertHandCount(gametest.PlayerA, "Firestorm Phoenix", 1) // returned to hand
	})
}

func TestShelkinBrownie(t *testing.T) {
	t.Run("removes banding from legendary creature with guildhouse", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Adventurers' Guildhouse")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Jasmine Boreal") // green legendary
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Shelkin Brownie")
		// Jasmine should have banding from the Guildhouse
		g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerB, "Shelkin Brownie", "Jasmine Boreal")
		g.StopAt(2, core.PostcombatMain)
		g.Execute()
		// Jasmine should have lost banding
		g.AssertHasAbility(gametest.PlayerA, "Jasmine Boreal", core.Banding, false)
	})
}

