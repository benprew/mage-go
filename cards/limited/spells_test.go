package limited

import (
	"testing"

	"github.com/mage/mage/pkg/mage/core"
	"github.com/mage/mage/pkg/mage/gametest"
)

// Tests for cards registered in alpha_spells.go.

func TestBalance(t *testing.T) {
	t.Run("equalizes_lands", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 4)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains", 2)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Balance")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Balance")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Balance: each player who controls more lands than the player who controls
		// the fewest sacrifices lands until all players control the same number.
		// A had 4, B had 2 -> A sacrifices 2, both end with 2.
		g.AssertPermanentCount(gametest.PlayerA, "Plains", 2)
		g.AssertPermanentCount(gametest.PlayerB, "Plains", 2)
	})

	t.Run("equalizes_creatures", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gray Ogre")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Savannah Lions")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Balance")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Balance")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// A had 3 creatures, B had 1 -> A sacrifices 2.
		playerA := g.AllPlayers()[0]
		creatureCount := 0
		for _, perm := range g.Battlefield {
			if perm.Controller == playerA.PlayerID() && perm.HasType(core.TypeCreature) {
				creatureCount++
			}
		}
		if creatureCount != 1 {
			t.Errorf("Balance should equalize creatures: PlayerA has %d creatures, want 1", creatureCount)
		}
	})

	t.Run("equalizes_hand_size", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Balance")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt", 4)
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt", 2)
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Balance")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// A had 5 cards (Balance + 4 Bolts), casts Balance (now 4 in hand), B has 2.
		// Balance resolves: A discards down to 2.
		playerA := g.AllPlayers()[0]
		playerB := g.AllPlayers()[1]
		if len(playerA.Hand()) != len(playerB.Hand()) {
			t.Errorf("Balance should equalize hand sizes: A has %d, B has %d",
				len(playerA.Hand()), len(playerB.Hand()))
		}
	})

	t.Run("does_nothing_when_equal", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Plains", 2)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains", 2)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Balance")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Balance")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Both players have equal counts -- nothing should change.
		g.AssertPermanentCount(gametest.PlayerA, "Plains", 2)
		g.AssertPermanentCount(gametest.PlayerB, "Plains", 2)
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 1)
	})
}

func TestHealingSalve(t *testing.T) {
	t.Run("gain_3_life", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.SetLife(gametest.PlayerA, 17)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Healing Salve")
		g.ChooseMode(gametest.PlayerA, 0) // choose "gain 3 life"
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Healing Salve", "PlayerA")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 20)
	})

	t.Run("prevent_3_damage_to_creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant") // 3/3
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Healing Salve")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.ChooseMode(gametest.PlayerA, 1) // choose "prevent 3 damage"
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Healing Salve", "Hill Giant")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Hill Giant")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 1)
	})

	t.Run("prevent_3_damage_to_player", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.SetLife(gametest.PlayerA, 20)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Healing Salve")
		g.ChooseMode(gametest.PlayerA, 1) // choose "prevent 3 damage"
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Healing Salve", "PlayerA")
		g.Attack(1, gametest.PlayerB, "Hill Giant")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerA, 20)
	})
}

func TestReverseDamage(t *testing.T) {
	t.Run("prevents_damage_and_gains_life", func(t *testing.T) {
		// Reverse Damage: The next time a source of your choice would deal damage
		// to you this turn, prevent that damage. You gain life equal to the damage
		// prevented this way.
		g := gametest.NewTestGame(t)
		g.SetLife(gametest.PlayerB, 20)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Craw Wurm") // 6/4
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Reverse Damage")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Reverse Damage")
		g.Attack(1, gametest.PlayerA, "Craw Wurm")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Reverse Damage prevents 6 combat damage and gains 6 life: 20 + 6 = 26.
		g.AssertLife(gametest.PlayerB, 26)
	})
}

func TestDeathWard(t *testing.T) {
	t.Run("regenerates_target_creature", func(t *testing.T) {
		// Death Ward: Regenerate target creature.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Death Ward")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Death Ward", "Grizzly Bears")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Lightning Bolt", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Death Ward should regenerate Bears -- they survive lethal damage.
		g.AssertPermanentCount(gametest.PlayerA, "Grizzly Bears", 1)
	})
}

func TestSleightOfMind(t *testing.T) {
	t.Run("changes_color_word_on_permanent", func(t *testing.T) {
		// Sleight of Mind: Change the text of target permanent by replacing
		// all instances of one color word with another.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Bog Wraith") // has swampwalk
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Sleight of Mind")
		// Change "swamp" -> "forest" on Bog Wraith
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Sleight of Mind", "Bog Wraith")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Bog Wraith should now have forestwalk instead of swampwalk.
		g.AssertHasAbility(gametest.PlayerA, "Bog Wraith", core.Forestwalk, true)
		g.AssertHasAbility(gametest.PlayerA, "Bog Wraith", core.Swampwalk, false)
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

func TestFork(t *testing.T) {
	t.Run("copies_instant_spell", func(t *testing.T) {
		// Fork: Copy target instant or sorcery spell, except that the copy is red.
		// You may choose new targets for the copy.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Fork")
		// Cast Lightning Bolt targeting PlayerB, then Fork targeting the Bolt.
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
		g.CastInResponseTo(gametest.PlayerA, "Fork")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Fork copies Lightning Bolt -> both resolve -> 3 + 3 = 6 damage.
		g.AssertLife(gametest.PlayerB, 14)
	})
}

func TestChannel(t *testing.T) {
	t.Run("pay_life_for_colorless_mana", func(t *testing.T) {
		// Channel: Until end of turn, any time you could activate a mana ability,
		// you may pay 1 life. If you do, add {C}.
		g := gametest.NewTestGame(t)
		g.SetLife(gametest.PlayerA, 20)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Channel")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Fireball")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Channel")
		// After Channel, pay 5 life for 5 colorless, then cast Fireball X=5.
		g.CastSpellWithX(1, core.PrecombatMain, gametest.PlayerA, "Fireball", 5, "PlayerB")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Channel should enable paying 5 life (20 -> 15) to fuel Fireball.
		g.AssertLife(gametest.PlayerA, 15)
	})

	t.Run("pay_life_for_generic_mana", func(t *testing.T) {
		// Channel should also pay generic mana costs from life (not just X).
		// Cast Channel, then cast a non-X spell with a generic cost.
		g := gametest.NewTestGame(t)
		g.SetLife(gametest.PlayerA, 20)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Channel")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Hill Giant") // {3}{R} = 3 generic + 1 red
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Channel")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Hill Giant")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Channel pays 3 generic from life (20 -> 17). Red is paid from auto-mana.
		g.AssertLife(gametest.PlayerA, 17)
		g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 1)
	})
}
