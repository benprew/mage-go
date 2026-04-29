package fourthedition

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"

	_ "git.sr.ht/~cdcarter/mage-go/cards/limited"
)

var _ = registerCreatures
var _ = registerArtifacts
var _ = registerEnchantments
var _ = registerLands
var _ = registerSpells

// ===== SIMPLE KEYWORD CREATURES =====

func TestBogImp(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Bog Imp")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Bog Imp", 1, 1)
	g.AssertHasAbility(gametest.PlayerA, "Bog Imp", core.Flying, true)
}

func TestLandLeeches(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Land Leeches")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Land Leeches", 2, 2)
	g.AssertHasAbility(gametest.PlayerA, "Land Leeches", core.FirstStrike, true)
}

func TestCarnivorousPlant(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Carnivorous Plant")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Carnivorous Plant", 4, 5)
	g.AssertHasAbility(gametest.PlayerA, "Carnivorous Plant", core.Defender, true)
}

func TestPikemen(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Pikemen")
	g.StopAt(1, core.PrecombatMain)
	g.Execute()
	g.AssertPowerToughness(gametest.PlayerA, "Pikemen", 1, 1)
	g.AssertHasAbility(gametest.PlayerA, "Pikemen", core.FirstStrike, true)
	g.AssertHasAbility(gametest.PlayerA, "Pikemen", core.Banding, true)
}

// ===== MANA ABILITIES =====

func TestSistersOfTheFlame(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sisters of the Flame")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Sisters of the Flame")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertTapped(gametest.PlayerA, "Sisters of the Flame", true)
	pool := g.AllPlayers()[0].ManaPool()
	if pool.CountProducedThisTurn(core.Red) < 1 {
		t.Errorf("expected at least 1 red mana produced, got %d", pool.CountProducedThisTurn(core.Red))
	}
}

func TestApprenticeWizard(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Apprentice Wizard")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Apprentice Wizard")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertTapped(gametest.PlayerA, "Apprentice Wizard", true)
	pool := g.AllPlayers()[0].ManaPool()
	if pool.CountProducedThisTurn(core.Colorless) < 3 {
		t.Errorf("expected at least 3 colorless mana produced, got %d", pool.CountProducedThisTurn(core.Colorless))
	}
}

// ===== REGENERATION =====

func TestGhostShipRegenerate(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ghost Ship")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
	g.AssertHasAbility(gametest.PlayerA, "Ghost Ship", core.Flying, true)
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Ghost Ship")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Ghost Ship")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Ghost Ship", 1)
}

func TestDiabolicMachineRegenerate(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Diabolic Machine")
	g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Diabolic Machine")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Diabolic Machine")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	g.AssertPermanentCount(gametest.PlayerA, "Diabolic Machine", 1)
}

// ===== BALL LIGHTNING =====

func TestBallLightning(t *testing.T) {
	t.Run("has trample and haste", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ball Lightning")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Ball Lightning", 6, 1)
		g.AssertHasAbility(gametest.PlayerA, "Ball Lightning", core.Trample, true)
		g.AssertHasAbility(gametest.PlayerA, "Ball Lightning", core.Haste, true)
	})

	t.Run("sacrificed at end step", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ball Lightning")
		g.StopAt(2, core.Upkeep)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Ball Lightning", 0)
	})

	t.Run("deals 6 trample damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Ball Lightning")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Ball Lightning")
		g.Block(1, gametest.PlayerB, "Grizzly Bears", "Ball Lightning")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Ball Lightning 6/1 blocked by Grizzly Bears 2/2
		// 2 damage assigned to Bears (lethal), 4 tramples through
		g.AssertLife(gametest.PlayerB, 16)
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
	})
}

// ===== MURK DWELLERS =====

func TestMurkDwellers(t *testing.T) {
	t.Run("gets +2/+0 when unblocked", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Murk Dwellers")
		g.Attack(1, gametest.PlayerA, "Murk Dwellers")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// 2/2 + 2/0 = 4 damage unblocked
		g.AssertLife(gametest.PlayerB, 16)
	})

	t.Run("no bonus when blocked", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Murk Dwellers")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.Attack(1, gametest.PlayerA, "Murk Dwellers")
		g.Block(1, gametest.PlayerB, "Hill Giant", "Murk Dwellers")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Blocked: no +2/+0 bonus. Murk Dwellers 2/2 vs Hill Giant 3/3.
		// Murk Dwellers dies, Hill Giant survives.
		g.AssertLife(gametest.PlayerB, 20)
		g.AssertPermanentCount(gametest.PlayerA, "Murk Dwellers", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 1)
	})
}

// ===== BROTHERS OF FIRE =====

func TestBrothersOfFire(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Brothers of Fire")
	g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Brothers of Fire", "PlayerB")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	// 1 damage to PlayerB, 1 damage to PlayerA (controller)
	g.AssertLife(gametest.PlayerB, 19)
	g.AssertLife(gametest.PlayerA, 19)
}

// ===== CAVE PEOPLE =====

func TestCavePeople(t *testing.T) {
	t.Run("gets +1/-2 when attacking", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Cave People")
		g.Attack(1, gametest.PlayerA, "Cave People")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// 1/4 + 1/-2 = 2/2, deals 2 damage
		g.AssertLife(gametest.PlayerB, 18)
	})

	t.Run("grants mountainwalk", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Cave People")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Cave People", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertHasAbility(gametest.PlayerA, "Grizzly Bears", core.Mountainwalk, true)
	})
}

// ===== UNCLE ISTVAN =====

func TestUncleIstvan(t *testing.T) {
	t.Run("prevents damage from creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Uncle Istvan")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.Attack(2, gametest.PlayerB, "Hill Giant")
		g.Block(2, gametest.PlayerA, "Uncle Istvan", "Hill Giant")
		g.StopAt(2, core.EndCombat)
		g.Execute()
		// Uncle Istvan 1/3 blocks Hill Giant 3/3
		// Hill Giant's damage is prevented; Uncle Istvan deals 1 to Hill Giant
		g.AssertPermanentCount(gametest.PlayerA, "Uncle Istvan", 1)
		g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 1)
	})

	t.Run("still takes damage from spells", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Uncle Istvan")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Uncle Istvan")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		// Lightning Bolt is not a creature, so damage is dealt normally
		// Uncle Istvan 1/3 takes 3 damage and dies
		g.AssertPermanentCount(gametest.PlayerA, "Uncle Istvan", 0)
	})
}

// ===== DINGUS EGG =====

func TestDingusEgg(t *testing.T) {
	t.Run("deals 2 when land destroyed", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dingus Egg")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Sinkhole")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Sinkhole", "Forest")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Forest destroyed, Dingus Egg deals 2 to Forest's controller (PlayerB)
		g.AssertLife(gametest.PlayerB, 18)
		g.AssertLife(gametest.PlayerA, 20)
	})

	t.Run("triggers for each land", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dingus Egg")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Armageddon")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Armageddon")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Two lands destroyed: PlayerA's Plains and PlayerB's Island
		// Dingus Egg deals 2 to each controller
		g.AssertLife(gametest.PlayerA, 18)
		g.AssertLife(gametest.PlayerB, 18)
	})
}

// ===== CREATURE BOND =====

func TestCreatureBond(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Creature Bond")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Creature Bond", "Hill Giant")
	// Now kill the Hill Giant
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Hill Giant")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	// Hill Giant (3/3) dies with Creature Bond attached
	// Creature Bond deals 3 (toughness) to Hill Giant's controller (PlayerB)
	g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 0)
	g.AssertLife(gametest.PlayerB, 17)
}

// ===== VENOM =====

func TestVenom(t *testing.T) {
	t.Run("destroys non-Wall blocker", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Venom")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Craw Wurm")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Venom", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Craw Wurm", "Grizzly Bears")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Grizzly Bears (2/2 with BasiliskTouch) blocked by Craw Wurm (6/4)
		// Grizzly Bears dies to combat damage (takes 6), Craw Wurm is destroyed by BasiliskTouch
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Craw Wurm", 0)
	})

	t.Run("does not destroy Wall blocker", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Venom")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wall of Stone")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Venom", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Wall of Stone", "Grizzly Bears")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Wall of Stone is a Wall, so BasiliskTouch doesn't destroy it
		g.AssertPermanentCount(gametest.PlayerB, "Wall of Stone", 1)
	})
}

// ===== FLOOD =====

func TestFlood(t *testing.T) {
	t.Run("taps creature without flying", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Flood")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.ActivateAbility(1, core.PrecombatMain, gametest.PlayerA, "Flood", "Hill Giant")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertTapped(gametest.PlayerB, "Hill Giant", true)
	})
}

// ===== SUNKEN CITY =====

func TestSunkenCity(t *testing.T) {
	t.Run("boosts blue creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sunken City")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Merfolk of the Pearl Trident")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island", 2)
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Merfolk (1/1 blue) gets +1/+1 = 2/2
		g.AssertPowerToughness(gametest.PlayerA, "Merfolk of the Pearl Trident", 2, 2)
		// Grizzly Bears (2/2 green) no boost
		g.AssertPowerToughness(gametest.PlayerA, "Grizzly Bears", 2, 2)
	})

	t.Run("sacrificed if upkeep not paid", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Sunken City")
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// No mana to pay {U}{U}, sacrificed
		g.AssertPermanentCount(gametest.PlayerA, "Sunken City", 0)
	})
}

// ===== FISSURE =====

func TestFissure(t *testing.T) {
	t.Run("destroys creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Fissure")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Fissure", "Hill Giant")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 0)
	})

	t.Run("destroys land", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Fissure")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Fissure", "Forest")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Forest", 0)
	})

	t.Run("can't be regenerated", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Drudge Skeletons")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Fissure")
		// Activate regeneration
		g.ActivateAbility(2, core.PrecombatMain, gametest.PlayerB, "Drudge Skeletons")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerA, "Fissure", "Drudge Skeletons")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		// Can't be regenerated
		g.AssertPermanentCount(gametest.PlayerB, "Drudge Skeletons", 0)
	})
}

// ===== INFERNO =====

func TestInferno(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Craw Wurm")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Inferno")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Inferno")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	// 6 damage to each creature and each player
	// Hill Giant (3/3) dies, Craw Wurm (6/4) dies
	g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 0)
	g.AssertPermanentCount(gametest.PlayerB, "Craw Wurm", 0)
	g.AssertLife(gametest.PlayerA, 14)
	g.AssertLife(gametest.PlayerB, 14)
}

// ===== MARSH GAS =====

func TestMarshGas(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Marsh Gas")
	g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Marsh Gas")
	g.StopAt(1, core.BeginCombat)
	g.Execute()
	// All creatures get -2/-0
	// Hill Giant: 3/3 -> 1/3
	g.AssertPowerToughness(gametest.PlayerA, "Hill Giant", 1, 3)
	// Grizzly Bears: 2/2 -> 0/2
	g.AssertPowerToughness(gametest.PlayerB, "Grizzly Bears", 0, 2)
}

// ===== MORALE =====

func TestMorale(t *testing.T) {
	g := gametest.NewTestGame(t)
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
	g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
	g.AddCard(core.ZoneHand, gametest.PlayerA, "Morale")
	g.Attack(1, gametest.PlayerA, "Grizzly Bears")
	// Cast Morale during combat to boost attacking creatures
	g.CastSpell(1, core.DeclareBlockers, gametest.PlayerA, "Morale")
	g.StopAt(1, core.EndCombat)
	g.Execute()
	// Grizzly Bears is attacking, gets +1/+1 = 3/3, deals 3 damage
	// Hill Giant is not attacking, no boost
	g.AssertLife(gametest.PlayerB, 17)
}

// ===== ANIMATE ARTIFACT =====

func TestAnimateArtifact(t *testing.T) {
	t.Run("turns noncreature artifact into creature with CMC P/T", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fellwar Stone") // CMC 2
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Animate Artifact")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Animate Artifact", "Fellwar Stone")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Fellwar Stone", 2, 2)
	})

	t.Run("animated artifact can attack", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Dingus Egg") // CMC 4
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Animate Artifact")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Animate Artifact", "Dingus Egg")
		g.Attack(1, gametest.PlayerA, "Dingus Egg")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 16)
	})

	t.Run("does not affect artifact that is already a creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Diabolic Machine") // 4/4 artifact creature
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Animate Artifact")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Animate Artifact", "Diabolic Machine")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Diabolic Machine is already a creature, so Animate Artifact doesn't change its P/T
		g.AssertPowerToughness(gametest.PlayerA, "Diabolic Machine", 4, 4)
	})

	t.Run("effect ends when aura is removed", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Fellwar Stone") // CMC 2
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Animate Artifact")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Disenchant")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Animate Artifact", "Fellwar Stone")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Disenchant", "Animate Artifact")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Aura destroyed — Fellwar Stone is no longer a creature
		g.AssertPermanentCount(gametest.PlayerA, "Fellwar Stone", 1)
	})
}

// ===== MARSH VIPER =====

func TestMarshViper(t *testing.T) {
	t.Run("combat damage to player gives two poison counters", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Marsh Viper")
		g.Attack(1, gametest.PlayerA, "Marsh Viper")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 19)
		g.AssertPoisonCounters(gametest.PlayerB, 2)
	})

	t.Run("damage to creature does not give poison", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Marsh Viper")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // 2/2
		g.Attack(2, gametest.PlayerA, "Marsh Viper")
		g.Block(2, gametest.PlayerB, "Grizzly Bears", "Marsh Viper")
		g.StopAt(2, core.EndStep)
		g.Execute()
		g.AssertPoisonCounters(gametest.PlayerB, 0)
	})
}

// ===== MANA CLASH =====

func TestManaClash(t *testing.T) {
	t.Run("both heads first flip ends with no damage", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Mana Clash")
		g.SetCoinFlipResults([]bool{true, true}) // you heads, opp heads
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Mana Clash", "PlayerB")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 20)
		g.AssertLife(gametest.PlayerB, 20)
	})

	t.Run("opponent tails takes damage then both heads stops", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Mana Clash")
		// round 1: you heads, opp tails (opp -1) → continue
		// round 2: you heads, opp heads → stop
		g.SetCoinFlipResults([]bool{true, false, true, true})
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Mana Clash", "PlayerB")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 20)
		g.AssertLife(gametest.PlayerB, 19)
	})

	t.Run("both tails then both heads", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Mana Clash")
		// round 1: both tails (each -1) → continue
		// round 2: both heads → stop
		g.SetCoinFlipResults([]bool{false, false, true, true})
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Mana Clash", "PlayerB")
		g.StopAt(1, core.EndStep)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 19)
		g.AssertLife(gametest.PlayerB, 19)
	})
}
