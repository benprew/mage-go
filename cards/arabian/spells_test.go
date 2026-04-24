package arabian

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/gametest"
)

func TestPiety(t *testing.T) {
	t.Run("blocking_creatures_get_plus_0_plus_3", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")                // 2/2 attacker
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Merfolk of the Pearl Trident") // 1/1 blocker
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Piety")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Merfolk of the Pearl Trident", "Grizzly Bears")
		// Cast after blockers are declared (FirstStrikeDamage runs after DeclareBlockers)
		g.CastSpell(1, core.FirstStrikeDamage, gametest.PlayerB, "Piety")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// 1/1 becomes 1/4 with Piety — survives blocking a 2/2
		g.AssertPermanentCount(gametest.PlayerB, "Merfolk of the Pearl Trident", 1)
	})

	t.Run("non_blocking_creatures_unaffected", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3, not blocking
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Piety")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		// Hill Giant does NOT block
		g.CastSpell(1, core.FirstStrikeDamage, gametest.PlayerB, "Piety")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Bears hits through unblocked — 2 damage to PlayerB
		g.AssertLife(gametest.PlayerB, 18)
	})
}

func TestArmyOfAllah(t *testing.T) {
	t.Run("attacking_creatures_get_plus_2_plus_0", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Army of Allah")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		// Cast after attackers declared
		g.CastSpell(1, core.DeclareBlockers, gametest.PlayerA, "Army of Allah")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// 2/2 becomes 4/2 with Army of Allah — deals 4 damage
		g.AssertLife(gametest.PlayerB, 16)
	})

	t.Run("non_attacking_creatures_unaffected", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2 attacker
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")    // 3/3 stays back
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Army of Allah")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.CastSpell(1, core.DeclareBlockers, gametest.PlayerA, "Army of Allah")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// Only Bears attacked (boosted to 4/2), Hill Giant stayed back
		g.AssertLife(gametest.PlayerB, 16)
		g.AssertPowerToughness(gametest.PlayerA, "Hill Giant", 3, 3)
	})
}

func TestDesertTwister(t *testing.T) {
	t.Run("destroys_creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Desert Twister")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Desert Twister", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerB, "Grizzly Bears", 1)
	})

	t.Run("destroys_enchantment", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Fishliver Oil")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Desert Twister")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Desert Twister", "Fishliver Oil")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerB, "Fishliver Oil", 1)
	})

	t.Run("destroys_land", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Desert Twister")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Forest")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Desert Twister", "Island")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertGraveyardCount(gametest.PlayerB, "Island", 1)
	})
}

func TestMetamorphosis(t *testing.T) {
	t.Run("sacrifices creature and adds mana equal to 1 plus CMC", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // CMC 2
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Metamorphosis")
		g.ChooseManaColor(gametest.PlayerA, core.Green) // choose green mana
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Metamorphosis")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Grizzly Bears (CMC 2) sacrificed → 1+2 = 3 green mana added
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
		pool := g.AllPlayers()[0].ManaPool()
		if pool.CountProducedThisTurn(core.Green) < 3 {
			t.Errorf("expected at least 3 green mana from Metamorphosis, got %d", pool.CountProducedThisTurn(core.Green))
		}
	})

	t.Run("can_choose_different_color", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // CMC 2
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Metamorphosis")
		g.ChooseManaColor(gametest.PlayerA, core.Red) // choose red mana
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Metamorphosis")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 0)
		pool := g.AllPlayers()[0].ManaPool()
		if pool.CountProducedThisTurn(core.Red) < 3 {
			t.Errorf("expected at least 3 red mana from Metamorphosis, got %d", pool.CountProducedThisTurn(core.Red))
		}
	})
}

func TestSandstorm(t *testing.T) {
	t.Run("deals_1_to_each_attacker", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Flying Men")    // 1/1
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Sandstorm")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears", "Flying Men")
		// Cast after attackers declared
		g.CastSpell(1, core.DeclareBlockers, gametest.PlayerB, "Sandstorm")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Flying Men (1/1) dies to 1 damage
		g.AssertPermanentCount(gametest.PlayerA, "Flying Men", 0)
		// Grizzly Bears (2/2) survives but took 1
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	})

	t.Run("non_attacking_creatures_unaffected", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // attacking
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Flying Men")    // defender's 1/1, not attacking
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Sandstorm")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Forest")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears")
		g.CastSpell(1, core.DeclareBlockers, gametest.PlayerB, "Sandstorm")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Defender's Flying Men unaffected
		g.AssertPermanentCount(gametest.PlayerB, "Flying Men", 1)
	})
}

func TestEyeForAnEye(t *testing.T) {
	t.Run("reflects_damage_to_sources_controller", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Eye for an Eye")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		// Cast Eye for an Eye in response to Lightning Bolt targeting PlayerB
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
		g.CastInResponseTo(gametest.PlayerB, "Eye for an Eye")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// PlayerB took 3 damage from Bolt
		g.AssertLife(gametest.PlayerB, 17)
		// PlayerA took 3 damage reflected by Eye for an Eye
		g.AssertLife(gametest.PlayerA, 17)
	})

	t.Run("reflects_combat_damage_from_chosen_source", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant") // 3/3
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Eye for an Eye")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains")
		// Cast Eye for an Eye before combat; ChoosePermanent selects Hill Giant
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Eye for an Eye")
		g.ChoosePermanent(gametest.PlayerB, "Hill Giant")
		g.Attack(1, gametest.PlayerA, "Hill Giant")
		g.StopAt(1, core.PostcombatMain)
		g.Execute()
		// PlayerB took 3 combat damage from Hill Giant
		g.AssertLife(gametest.PlayerB, 17)
		// PlayerA took 3 reflected damage
		g.AssertLife(gametest.PlayerA, 17)
	})
}
