package cards

import (
	"testing"

	"github.com/mage/mage"
)

// Tests for cards registered in alpha_spells.go.

func TestBalance(t *testing.T) {
	t.Run("equalizes_lands", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Plains", 4)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Plains", 2)
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Balance")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Balance")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Balance: each player who controls more lands than the player who controls
		// the fewest sacrifices lands until all players control the same number.
		// A had 4, B had 2 -> A sacrifices 2, both end with 2.
		g.AssertPermanentCount(mage.PlayerA, "Plains", 2)
		g.AssertPermanentCount(mage.PlayerB, "Plains", 2)
	})

	t.Run("equalizes_creatures", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Hill Giant")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Gray Ogre")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Savannah Lions")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Balance")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Balance")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// A had 3 creatures, B had 1 -> A sacrifices 2.
		playerA := g.Players[0]
		creatureCount := 0
		for _, perm := range g.Battlefield {
			if perm.Controller == playerA.PlayerID() && perm.HasType(mage.TypeCreature) {
				creatureCount++
			}
		}
		if creatureCount != 1 {
			t.Errorf("Balance should equalize creatures: PlayerA has %d creatures, want 1", creatureCount)
		}
	})

	t.Run("equalizes_hand_size", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Balance")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Lightning Bolt", 4)
		g.AddCard(mage.ZoneHand, mage.PlayerB, "Lightning Bolt", 2)
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Balance")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// A had 5 cards (Balance + 4 Bolts), casts Balance (now 4 in hand), B has 2.
		// Balance resolves: A discards down to 2.
		playerA := g.Players[0]
		playerB := g.Players[1]
		if len(playerA.Hand()) != len(playerB.Hand()) {
			t.Errorf("Balance should equalize hand sizes: A has %d, B has %d",
				len(playerA.Hand()), len(playerB.Hand()))
		}
	})

	t.Run("does_nothing_when_equal", func(t *testing.T) {
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Plains", 2)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Plains", 2)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerB, "Grizzly Bears")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Balance")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Balance")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Both players have equal counts -- nothing should change.
		g.AssertPermanentCount(mage.PlayerA, "Plains", 2)
		g.AssertPermanentCount(mage.PlayerB, "Plains", 2)
		g.AssertPermanentCount(mage.PlayerA, "Grizzly Bears", 1)
		g.AssertPermanentCount(mage.PlayerB, "Grizzly Bears", 1)
	})
}

func TestHealingSalve(t *testing.T) {
	t.Run("prevents_3_damage", func(t *testing.T) {
		// Healing Salve mode 2: Prevent the next 3 damage that would be dealt
		// to any target this turn. (The gain-3-life mode already works.)
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Hill Giant") // 3/3
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Healing Salve")
		g.AddCard(mage.ZoneHand, mage.PlayerB, "Lightning Bolt")
		// Cast Healing Salve in prevent mode targeting Hill Giant
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Healing Salve", "Hill Giant")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerB, "Lightning Bolt", "Hill Giant")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// 3 damage from Bolt prevented -> Hill Giant survives.
		g.AssertPermanentCount(mage.PlayerA, "Hill Giant", 1)
	})
}

func TestReverseDamage(t *testing.T) {
	t.Run("prevents_damage_and_gains_life", func(t *testing.T) {
		// Reverse Damage: The next time a source of your choice would deal damage
		// to you this turn, prevent that damage. You gain life equal to the damage
		// prevented this way.
		g := mage.NewTestGame(t)
		g.SetLife(mage.PlayerB, 20)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Craw Wurm") // 6/4
		g.AddCard(mage.ZoneHand, mage.PlayerB, "Reverse Damage")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerB, "Reverse Damage")
		g.Attack(1, mage.PlayerA, "Craw Wurm")
		g.StopAt(1, mage.EndCombat)
		g.Execute()
		// Reverse Damage prevents 6 combat damage and gains 6 life: 20 + 6 = 26.
		g.AssertLife(mage.PlayerB, 26)
	})
}

func TestDeathWard(t *testing.T) {
	t.Run("regenerates_target_creature", func(t *testing.T) {
		// Death Ward: Regenerate target creature.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears") // 2/2
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Death Ward")
		g.AddCard(mage.ZoneHand, mage.PlayerB, "Lightning Bolt")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Death Ward", "Grizzly Bears")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerB, "Lightning Bolt", "Grizzly Bears")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Death Ward should regenerate Bears -- they survive lethal damage.
		g.AssertPermanentCount(mage.PlayerA, "Grizzly Bears", 1)
	})
}

func TestSleightOfMind(t *testing.T) {
	t.Run("changes_color_word_on_permanent", func(t *testing.T) {
		// Sleight of Mind: Change the text of target permanent by replacing
		// all instances of one color word with another.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Bog Wraith") // has swampwalk
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Sleight of Mind")
		// Change "swamp" -> "forest" on Bog Wraith
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Sleight of Mind", "Bog Wraith")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Bog Wraith should now have forestwalk instead of swampwalk.
		g.AssertHasAbility(mage.PlayerA, "Bog Wraith", mage.Forestwalk, true)
		g.AssertHasAbility(mage.PlayerA, "Bog Wraith", mage.Swampwalk, false)
	})
}

func TestStasis(t *testing.T) {
	t.Run("players_skip_untap_step", func(t *testing.T) {
		// Stasis: Players skip their untap steps.
		g := mage.NewTestGame(t)
		// Island lets PlayerA pay {U} at turn 1 upkeep so Stasis survives.
		// At turn 3 untap the Island is tapped (used for payment) and can't
		// untap (Stasis prevents it), so Stasis is sacrificed at turn 3 upkeep.
		// But the untap prevention already happened, so Bears stay tapped.
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Island")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Stasis")
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Grizzly Bears")
		// Attack with Bears to tap them
		g.Attack(1, mage.PlayerA, "Grizzly Bears")
		// Turn 3 is PlayerA's next turn -- Bears should NOT untap with Stasis.
		g.StopAt(3, mage.PrecombatMain)
		g.Execute()
		g.AssertTapped(mage.PlayerA, "Grizzly Bears", true)
	})

	t.Run("sacrifice_unless_pay_U", func(t *testing.T) {
		// At the beginning of your upkeep, sacrifice Stasis unless you pay {U}.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneBattlefield, mage.PlayerA, "Stasis")
		g.StopAt(1, mage.PrecombatMain)
		g.Execute()
		// Stasis should be sacrificed at upkeep (no {U} paid).
		g.AssertPermanentCount(mage.PlayerA, "Stasis", 0)
	})
}

func TestFork(t *testing.T) {
	t.Run("copies_instant_spell", func(t *testing.T) {
		// Fork: Copy target instant or sorcery spell, except that the copy is red.
		// You may choose new targets for the copy.
		g := mage.NewTestGame(t)
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Lightning Bolt")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Fork")
		// Cast Lightning Bolt targeting PlayerB, then Fork targeting the Bolt.
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Lightning Bolt", "PlayerB")
		g.CastInResponseTo(mage.PlayerA, "Fork")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Fork copies Lightning Bolt -> both resolve -> 3 + 3 = 6 damage.
		g.AssertLife(mage.PlayerB, 14)
	})
}

func TestChannel(t *testing.T) {
	t.Run("pay_life_for_colorless_mana", func(t *testing.T) {
		// Channel: Until end of turn, any time you could activate a mana ability,
		// you may pay 1 life. If you do, add {C}.
		g := mage.NewTestGame(t)
		g.SetLife(mage.PlayerA, 20)
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Channel")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Fireball")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Channel")
		// After Channel, pay 5 life for 5 colorless, then cast Fireball X=5.
		g.CastSpellWithX(1, mage.PrecombatMain, mage.PlayerA, "Fireball", 5, "PlayerB")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Channel should enable paying 5 life (20 -> 15) to fuel Fireball.
		g.AssertLife(mage.PlayerA, 15)
	})

	t.Run("pay_life_for_generic_mana", func(t *testing.T) {
		// Channel should also pay generic mana costs from life (not just X).
		// Cast Channel, then cast a non-X spell with a generic cost.
		g := mage.NewTestGame(t)
		g.SetLife(mage.PlayerA, 20)
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Channel")
		g.AddCard(mage.ZoneHand, mage.PlayerA, "Hill Giant") // {3}{R} = 3 generic + 1 red
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Channel")
		g.CastSpell(1, mage.PrecombatMain, mage.PlayerA, "Hill Giant")
		g.StopAt(1, mage.BeginCombat)
		g.Execute()
		// Channel pays 3 generic from life (20 -> 17). Red is paid from auto-mana.
		g.AssertLife(mage.PlayerA, 17)
		g.AssertPermanentCount(mage.PlayerA, "Hill Giant", 1)
	})
}
