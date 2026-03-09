package arabian

import (
	"testing"

	"github.com/mage/mage/pkg/mage/core"
	"github.com/mage/mage/pkg/mage/gametest"
)

func TestMagneticMountain(t *testing.T) {
	t.Run("blue_creature_doesnt_untap", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Magnetic Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Flying Men") // blue 1/1
		// Attack to tap Flying Men
		g.Attack(1, gametest.PlayerA, "Flying Men")
		// Turn 3 is PlayerA's next turn — blue creature should NOT untap
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Flying Men", true)
	})

	t.Run("blue_creature_untaps_if_pay_4", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Magnetic Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Flying Men") // blue 1/1
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Mountain")
		g.Attack(1, gametest.PlayerA, "Flying Men")
		// Turn 3 upkeep: pay {4} to untap
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Flying Men", false)
	})

	t.Run("non_blue_creature_unaffected", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Magnetic Mountain")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // green 2/2
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		// Turn 3: non-blue creature should untap normally
		g.StopAt(3, core.PrecombatMain)
		g.Execute()
		g.AssertTapped(gametest.PlayerA, "Grizzly Bears", false)
	})
}

func TestFishliverOil(t *testing.T) {
	t.Run("grants_islandwalk", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Fishliver Oil")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wall of Stone") // 0/8 Defender
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Fishliver Oil", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Wall of Stone", "Grizzly Bears")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// Islandwalk — can't be blocked (defender has Island)
		g.AssertLife(gametest.PlayerB, 18)
	})

	t.Run("blockable_without_island", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Fishliver Oil")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Wall of Stone")
		// No Island for defender
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Fishliver Oil", "Grizzly Bears")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Wall of Stone", "Grizzly Bears")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// Can be blocked normally
		g.AssertLife(gametest.PlayerB, 20)
	})
}

func TestUnstableMutation(t *testing.T) {
	t.Run("immediate_boost", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Flying Men") // 1/1
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Unstable Mutation")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Unstable Mutation", "Flying Men")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPowerToughness(gametest.PlayerA, "Flying Men", 4, 4) // 1+3, 1+3
	})

	t.Run("after_one_upkeep_gets_counter", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Flying Men") // 1/1
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Unstable Mutation")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Island")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Unstable Mutation", "Flying Men")
		g.StopAt(3, core.PrecombatMain) // next PlayerA upkeep: turn 3
		g.Execute()
		// +3/+3 from mutation, -1/-1 from one counter = net +2/+2
		g.AssertPowerToughness(gametest.PlayerA, "Flying Men", 3, 3)
	})
}

func TestDropOfHoney(t *testing.T) {
	t.Run("destroys_least_power_creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Drop of Honey")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")    // 3/3
		// At PlayerA's upkeep (turn 1), destroy least power creature
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// Grizzly Bears (power 2) should be destroyed, Hill Giant (power 3) survives
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 1)
	})

	t.Run("sacrifices_self_when_no_creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Drop of Honey")
		// No creatures on battlefield — Drop of Honey should sacrifice itself
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Drop of Honey", 0)
	})

	t.Run("sacrifices_self_when_last_creature_dies_mid_turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Drop of Honey")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")    // 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Flying Men")       // 1/1
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		// Turn 1 upkeep: destroys Flying Men (least power = 1)
		// Then cast Lightning Bolt on Grizzly Bears — last creature dies
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Both creatures dead, Drop of Honey should sacrifice itself
		g.AssertPermanentCount(gametest.PlayerA, "Drop of Honey", 0)
	})
}

func TestCyclone(t *testing.T) {
	t.Run("adds_counter_and_deals_damage_if_paid", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Cyclone")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest") // to pay {G}
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")
		// Turn 1 upkeep: add 1 wind counter, pay {G} → deal 1 to all creatures/players
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		// 1 damage to each creature and player
		g.AssertLife(gametest.PlayerA, 19)
		g.AssertLife(gametest.PlayerB, 19)
		// Cyclone should still be on the battlefield
		g.AssertPermanentCount(gametest.PlayerA, "Cyclone", 1)
	})

	t.Run("sacrifices_if_cant_pay", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Cyclone")
		// No forests to pay — Cyclone should be sacrificed
		g.StopAt(1, core.PrecombatMain)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Cyclone", 0)
	})
}

func TestJihad(t *testing.T) {
	t.Run("boosts_white_creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Savannah Lions") // 2/1 white
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")  // green permanent
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Jihad")
		g.ChooseManaColor(gametest.PlayerA, core.Green) // Jihad ETB: choose Green
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Jihad")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Chose Green, opponent has Grizzly Bears (green) → white creatures get +2/+1
		g.AssertPowerToughness(gametest.PlayerA, "Savannah Lions", 4, 2)
	})

	t.Run("sacrifices_when_condition_fails", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Jihad")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		g.ChooseManaColor(gametest.PlayerA, core.Green) // Jihad ETB: choose Green
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Jihad")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Grizzly Bears destroyed → opponent has no green permanents → Jihad sacrificed
		g.AssertPermanentCount(gametest.PlayerA, "Jihad", 0)
	})
}

func TestOubliette(t *testing.T) {
	t.Run("phases_out_creature_on_etb", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Oubliette")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Oubliette", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Grizzly Bears should be phased out (invisible)
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertPermanentCount(gametest.PlayerA, "Oubliette", 1)
		// Not in exile — it's phased out on the battlefield
		g.AssertExileCount("Grizzly Bears", 0)
	})

	t.Run("returns_creature_when_oubliette_destroyed", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Oubliette")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Desert Twister")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Oubliette", "Grizzly Bears")
		// Destroy Oubliette — creature should phase back in
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Desert Twister", "Oubliette")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Grizzly Bears should be back on the battlefield (tapped)
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
		g.AssertTapped(gametest.PlayerB, "Grizzly Bears", true)
		g.AssertPermanentCount(gametest.PlayerA, "Oubliette", 0)
	})

	t.Run("phasing_preserves_counters", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCounters(1, core.PrecombatMain, gametest.PlayerB, "Grizzly Bears", core.P1P1, 2)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Oubliette")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Desert Twister")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Oubliette", "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Desert Twister", "Oubliette")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Bears return with counters intact (phasing preserves them)
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
		g.AssertCounterCount(gametest.PlayerB, "Grizzly Bears", core.P1P1, 2)
	})
}
