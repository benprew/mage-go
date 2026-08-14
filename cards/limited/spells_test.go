package limited

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
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
		for _, perm := range g.AllBattlefield() {
			if perm.ControllerID() == playerA.PlayerID() && perm.HasType(core.TypeCreature) {
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
		g.ChooseTarget(gametest.PlayerB, "Craw Wurm")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerB, "Reverse Damage")
		g.Attack(1, gametest.PlayerA, "Craw Wurm")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Reverse Damage prevents 6 combat damage and gains 6 life: 20 + 6 = 26.
		g.AssertLife(gametest.PlayerB, 26)
	})

	t.Run("does_not_reverse_damage_from_previous_turn", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Craw Wurm")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Reverse Damage")
		g.Attack(1, gametest.PlayerA, "Craw Wurm")
		g.ChooseTarget(gametest.PlayerB, "Craw Wurm")
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerB, "Reverse Damage")
		g.StopAt(2, core.BeginCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 14)
	})

	t.Run("only_reverses_damage_from_the_chosen_source", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Craw Wurm")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Reverse Damage")
		g.ChooseTarget(gametest.PlayerB, "Craw Wurm")
		g.CastSpell(1, core.Upkeep, gametest.PlayerB, "Reverse Damage")
		g.Attack(1, gametest.PlayerA, "Hill Giant", "Craw Wurm")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 23)
	})

	t.Run("can_choose_a_spell_on_the_stack", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Lightning Bolt")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Reverse Damage")
		g.ChooseTarget(gametest.PlayerB, "Lightning Bolt")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Lightning Bolt", "PlayerB")
		g.CastInResponseTo(gametest.PlayerB, "Reverse Damage")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 23)
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

func TestShock(t *testing.T) {
	t.Run("deals_2_to_creature", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Shock")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Shock", "Grizzly Bears")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
	})

	t.Run("deals_2_to_player", func(t *testing.T) {
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Shock")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Shock", "PlayerB")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		g.AssertLife(gametest.PlayerB, 18)
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

func TestSacrifice(t *testing.T) {
	t.Run("sacrifice_creature_add_mana", func(t *testing.T) {
		// Sacrifice: As an additional cost to cast this spell, sacrifice a creature.
		// Add an amount of {B} equal to the sacrificed creature's mana value.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant") // CMC 4
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Sacrifice")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Sacrifice", "Hill Giant")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Hill Giant (CMC 4) sacrificed -> add {B}{B}{B}{B}.
		g.AssertPermanentCount(gametest.PlayerA, "Hill Giant", 0)
		pool := g.AllPlayers()[0].ManaPool()
		if pool.CountProducedThisTurn(core.Black) < 4 { // 4 from sacrificed Hill Giant (CMC 4)
			t.Errorf("Sacrifice should add 4 black (CMC of Hill Giant); expected >= 4 black, got %d", pool.CountProducedThisTurn(core.Black))
		}
	})
}

func TestWordOfCommand(t *testing.T) {
	t.Run("look_at_opponent_hand_and_play_card", func(t *testing.T) {
		// Word of Command: Look at target opponent's hand and choose a card from
		// it. You control that player until Word of Command finishes resolving.
		// The player plays that card if able.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Word of Command")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Word of Command", "PlayerB")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Word of Command should force PlayerB to cast Lightning Bolt.
		g.AssertHandCount(gametest.PlayerB, "Lightning Bolt", 0)
	})
}

func TestCamouflage(t *testing.T) {
	t.Run("randomizes_blocking", func(t *testing.T) {
		// Camouflage: Cast only during your declare attackers step. This turn,
		// instead of the defending player choosing blockers, you assign each
		// creature the defending player controls to block attacking creatures.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Savannah Lions")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Camouflage")
		g.CastSpell(1, core.DeclareAttackers, gametest.PlayerA, "Camouflage")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears", "Hill Giant")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Camouflage should let the attacker assign blockers.
		// At least one attacker should deal damage (5 total unblocked).
		g.AssertLife(gametest.PlayerB, 15) // 2 + 3 = 5 damage
	})
}

func TestNaturalSelection(t *testing.T) {
	t.Run("shuffles_library", func(t *testing.T) {
		// Natural Selection: Look at the top 3 cards of target player's library,
		// then put them back in any order. You may have that player shuffle.
		// In an automated engine, the rearrange has no effect, so we always shuffle.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Natural Selection")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Lightning Bolt")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Hill Giant")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Craw Wurm")
		g.AddCard(core.ZoneLibrary, gametest.PlayerA, "Forest")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Natural Selection", "PlayerA")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Library should still have all 5 cards (shuffle doesn't lose cards)
		playerA := g.AllPlayers()[0]
		if len(playerA.Library()) < 5 {
			t.Errorf("Natural Selection should preserve all library cards; got %d, want >= 5", len(playerA.Library()))
		}
	})
}

func TestManaShort(t *testing.T) {
	t.Run("taps_all_lands_and_empties_pool", func(t *testing.T) {
		// Mana Short: Tap all lands target player controls and that player loses
		// all unspent mana.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Island")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Mana Short")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Mana Short", "PlayerB")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// All of PlayerB's lands should be tapped.
		g.AssertTapped(gametest.PlayerB, "Plains", true)
		g.AssertTapped(gametest.PlayerB, "Island", true)
	})
}

func TestDrainPower(t *testing.T) {
	t.Run("steals_opponent_mana", func(t *testing.T) {
		// Drain Power: Target player activates a mana ability of each land they
		// control. Then that player loses all unspent mana and you add the mana
		// lost this way.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Plains")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Mountain")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Drain Power")
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Drain Power", "PlayerB")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// PlayerB's lands tapped, PlayerA gets the mana.
		g.AssertTapped(gametest.PlayerB, "Plains", true)
		g.AssertTapped(gametest.PlayerB, "Mountain", true)
	})
}

func TestSimulacrum(t *testing.T) {
	t.Run("gain_life_and_deal_damage", func(t *testing.T) {
		// Simulacrum: You gain life equal to the damage dealt to you this turn.
		// Simulacrum deals damage to target creature you control equal to the
		// damage dealt to you this turn.
		g := gametest.NewTestGame(t)
		g.SetLife(gametest.PlayerA, 17)                                    // took 3 damage earlier this "turn"
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Hill Giant")    // 3/3
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // 2/2
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Lightning Bolt")
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Simulacrum")
		// PlayerA casts Simulacrum; PlayerB bolts in response so the
		// 3 damage lands before Simulacrum resolves (CR 117.1b).
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Simulacrum", "Hill Giant")
		g.CastInResponseTo(gametest.PlayerB, "Lightning Bolt", "PlayerA")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Took 3 damage this turn -> gain 3 life (14 -> 17) and deal 3 to Hill Giant.
		g.AssertLife(gametest.PlayerA, 17)
	})
}

func TestBlazeOfGlory(t *testing.T) {
	t.Run("target_blocks_all_attackers", func(t *testing.T) {
		// Blaze of Glory: Target creature can block any number of creatures this
		// turn and must block each attacking creature if able.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Grizzly Bears")
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Gray Ogre")
		g.AddCard(core.ZoneHand, gametest.PlayerB, "Blaze of Glory")
		g.CastSpell(1, core.DeclareAttackers, gametest.PlayerB, "Blaze of Glory", "Hill Giant")
		g.Attack(1, gametest.PlayerA, "Grizzly Bears", "Gray Ogre")
		g.Block(1, gametest.PlayerB, "Hill Giant", "Grizzly Bears")
		g.Block(1, gametest.PlayerB, "Hill Giant", "Gray Ogre")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Hill Giant blocks both attackers (2 + 2 = 4 damage, lethal to 3/3).
		// With Blaze of Glory enabling multi-block, no damage gets through.
		g.AssertLife(gametest.PlayerB, 20)
	})
}

func TestFalseOrders(t *testing.T) {
	t.Run("removes_blocker_and_reassigns", func(t *testing.T) {
		// False Orders: Cast only during combat after blockers are declared.
		// Remove target creature defending player controls from combat.
		// Attacker(s) it blocked that other creatures blocked become unblocked.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Craw Wurm")  // 6/4
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant") // 3/3 blocker
		g.AddCard(core.ZoneHand, gametest.PlayerA, "False Orders")
		g.Attack(1, gametest.PlayerA, "Craw Wurm")
		g.Block(1, gametest.PlayerB, "Hill Giant", "Craw Wurm")
		// Cast False Orders after blocks to remove Hill Giant from combat.
		g.CastSpell(1, core.DeclareBlockers, gametest.PlayerA, "False Orders", "Hill Giant")
		g.StopAt(1, core.EndCombat)
		g.Execute()
		// Hill Giant removed from combat -> Craw Wurm becomes unblocked -> 6 damage.
		g.AssertLife(gametest.PlayerB, 14)
	})
}

func TestSirensCall(t *testing.T) {
	t.Run("forces_creatures_to_attack", func(t *testing.T) {
		// Siren's Call: Cast only during an opponent's turn, before attackers
		// are declared. Creatures the active player controls attack this turn if
		// able. At end of turn, destroy all non-Wall creatures that player
		// controls that didn't attack. Ignore this effect for each creature the
		// player didn't control continuously since the beginning of the turn.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Grizzly Bears") // 2/2
		g.AddCard(core.ZoneBattlefield, gametest.PlayerB, "Hill Giant")    // 3/3
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Siren's Call")
		// Cast on PlayerB's turn before attackers.
		g.CastSpell(2, core.PrecombatMain, gametest.PlayerA, "Siren's Call")
		// PlayerB doesn't attack with either creature.
		g.StopAt(2, core.Cleanup)
		g.Execute()
		// Non-attackers should be destroyed at end of turn.
		g.AssertPermanentCount(gametest.PlayerB, "Grizzly Bears", 0)
		g.AssertPermanentCount(gametest.PlayerB, "Hill Giant", 0)
	})
}

func TestMagicalHack(t *testing.T) {
	t.Run("changes_land_type_word", func(t *testing.T) {
		// Magical Hack: Change the text of target permanent by replacing all
		// instances of one basic land type with another.
		g := gametest.NewTestGame(t)
		g.AddCard(core.ZoneBattlefield, gametest.PlayerA, "Bog Wraith") // has swampwalk
		g.AddCard(core.ZoneHand, gametest.PlayerA, "Magical Hack")
		// Change "swamp" -> "forest" on Bog Wraith.
		g.CastSpell(1, core.PrecombatMain, gametest.PlayerA, "Magical Hack", "Bog Wraith")
		g.StopAt(1, core.BeginCombat)
		g.Execute()
		// Bog Wraith should have forestwalk instead of swampwalk.
		g.AssertHasAbility(gametest.PlayerA, "Bog Wraith", core.Forestwalk, true)
		g.AssertHasAbility(gametest.PlayerA, "Bog Wraith", core.Swampwalk, false)
	})
}
