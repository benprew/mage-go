// Package gametest: turn-structure tests for combat damage. Covers CR 510.1b
// (multiple unblocked attackers), CR 510.1c (damage assignment among multiple
// blockers), CR 510.2 (simultaneous combat damage), CR 510.3a (post-damage
// triggers resolve before post-damage priority), and CR 510.4 (first and
// double strike damage steps).
package gametest

import (
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// TestCombatDamage covers CR 510 sub-rules that govern combat damage
// assignment and the first-strike / regular damage steps.
func TestCombatDamage(t *testing.T) {

	// CR 510.2 — Each creature that's been dealt lethal damage is destroyed
	// as a state-based action. Combat damage is dealt simultaneously, so two
	// 2/2s blocking each other both die.
	t.Run("CR 510.2 combat damage is simultaneous", func(t *testing.T) {
		bearA := "Combat Sim Bear A"
		bearB := "Combat Sim Bear B"
		for _, n := range []string{bearA, bearB} {

			if !mage.CardRegistered(n) {
				mage.Register(n, func() mage.Card {
					return mage.NewCreature(n, "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
				})
			}
		}
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, bearA)
		tg.AddCard(core.ZoneBattlefield, PlayerB, bearB)
		tg.Attack(1, PlayerA, bearA)
		tg.Block(1, PlayerB, bearB, bearA)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		tg.AssertPermanentCount(PlayerA, bearA, 0)
		tg.AssertPermanentCount(PlayerB, bearB, 0)
		tg.AssertGraveyardCount(PlayerA, bearA, 1)
		tg.AssertGraveyardCount(PlayerB, bearB, 1)
	})

	// CR 510.4 — If at least one attacking or blocking creature has first
	// strike or double strike as the combat damage step begins, the only
	// creatures that assign combat damage in that step are those with first
	// strike or double strike. After that step, there's a second combat damage
	// step during which the rest deal combat damage.
	//
	// A 2/2 with first strike vs a 2/2 vanilla: first-striker kills the other
	// before it can deal combat damage, so the first-striker survives.
	t.Run("CR 510.4 first strike kills before normal damage", func(t *testing.T) {
		fs := "Combat FS 2/2"
		vn := "Combat Vanilla 2/2"
		if !mage.CardRegistered(fs) {
			mage.Register(fs, func() mage.Card {
				return mage.NewCreature(fs, "{1}{W}", 2, 2,
					mage.WithSubTypes("Soldier"),
					mage.WithKeyword(core.FirstStrike))
			})
		}
		if !mage.CardRegistered(vn) {
			mage.Register(vn, func() mage.Card {
				return mage.NewCreature(vn, "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
			})
		}
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, fs)
		tg.AddCard(core.ZoneBattlefield, PlayerB, vn)
		tg.Attack(1, PlayerA, fs)
		tg.Block(1, PlayerB, vn, fs)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		tg.AssertPermanentCount(PlayerA, fs, 1) // first striker survived
		tg.AssertPermanentCount(PlayerB, vn, 0) // vanilla died in first-strike step
		tg.AssertGraveyardCount(PlayerB, vn, 1)
	})

	// CR 510.4 — Double strike creates the same first-strike window and also
	// deals damage in the regular combat damage step.
	//
	// A 2/2 with double strike vs a 4/4 vanilla: the double-striker deals 2 in
	// the first-strike step (4/4 takes 2), then 2 more in the regular combat
	// damage step (4/4 now lethal), killing the 4/4. The 4/4 never deals
	// damage because it dies before the regular combat damage step.
	t.Run("CR 510.4 double strike deals damage in both damage steps", func(t *testing.T) {
		ds := "Combat DS 2/2"
		big := "Combat Vanilla 4/4"
		if !mage.CardRegistered(ds) {
			mage.Register(ds, func() mage.Card {
				return mage.NewCreature(ds, "{1}{R}{W}", 2, 2,
					mage.WithSubTypes("Soldier"),
					mage.WithKeyword(core.DoubleStrike))
			})
		}
		if !mage.CardRegistered(big) {
			mage.Register(big, func() mage.Card {
				return mage.NewCreature(big, "{2}{G}{G}", 4, 4, mage.WithSubTypes("Beast"))
			})
		}
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, ds)
		tg.AddCard(core.ZoneBattlefield, PlayerB, big)
		tg.Attack(1, PlayerA, ds)
		tg.Block(1, PlayerB, big, ds)
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		// The 4/4 dying to a 2/2 is the observable proof that damage was dealt
		// in both the first-strike step (2) and the regular combat damage
		// step (2); a single-striker 2/2 could never kill a 4/4.
		tg.AssertPermanentCount(PlayerB, big, 0)
		tg.AssertGraveyardCount(PlayerB, big, 1)
	})

	// CR 510.1b — An unblocked attacking creature assigns its combat damage to
	// the player or planeswalker it's attacking. With multiple unblocked
	// attackers, the defending player takes the sum of all their powers.
	t.Run("CR 510.1b multiple unblocked attackers sum damage to defender", func(t *testing.T) {
		a := "510.1b Attacker 2/2"
		b := "510.1b Attacker 3/3"
		if !mage.CardRegistered(a) {
			mage.Register(a, func() mage.Card {
				return mage.NewCreature(a, "{1}{G}", 2, 2, mage.WithSubTypes("Bear"))
			})
		}
		if !mage.CardRegistered(b) {
			mage.Register(b, func() mage.Card {
				return mage.NewCreature(b, "{2}{G}", 3, 3, mage.WithSubTypes("Bear"))
			})
		}
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, a)
		tg.AddCard(core.ZoneBattlefield, PlayerA, b)
		tg.Attack(1, PlayerA, a, b)
		tg.StopAt(1, core.PostcombatMain)
		tg.Execute()
		// 2 + 3 = 5 damage to defender.
		tg.AssertLife(PlayerB, 15)
	})

	// CR 510.1c — If an attacking creature is blocked by multiple creatures,
	// the attacking creature's controller chooses the damage assignment order
	// among the blockers and divides the attacker's combat damage among them,
	// subject to lethal-damage-first constraints. A 4/4 blocked by two 2/3s
	// should be able to assign 4 to exactly one blocker, killing it, and 0 to
	// the other (the first blocker already took lethal).
	t.Run("CR 510.1c controller divides damage among multiple blockers", func(t *testing.T) {
		// 4/4 attacker blocked by two 2/3s. Attacker's controller orders the
		// blockers and assigns 3 (lethal) to the first and 1 to the second.
		// First blocker dies; second survives with 1 damage marked. The 4/4
		// also takes 2+2=4 simultaneous damage from the blockers and dies.
		atk := "510.1c Attacker 4/4"
		blkA := "510.1c Blocker 2/3 A"
		blkB := "510.1c Blocker 2/3 B"
		if !mage.CardRegistered(atk) {
			mage.Register(atk, func() mage.Card {
				return mage.NewCreature(atk, "{2}{G}{G}", 4, 4, mage.WithSubTypes("Beast"))
			})
		}
		for _, n := range []string{blkA, blkB} {

			if !mage.CardRegistered(n) {
				mage.Register(n, func() mage.Card {
					return mage.NewCreature(n, "{1}{W}{W}", 2, 3, mage.WithSubTypes("Soldier"))
				})
			}
		}
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, atk)
		tg.AddCard(core.ZoneBattlefield, PlayerB, blkA)
		tg.AddCard(core.ZoneBattlefield, PlayerB, blkB)
		tg.Attack(1, PlayerA, atk)
		tg.Block(1, PlayerB, blkA, atk)
		tg.Block(1, PlayerB, blkB, atk)
		tg.ChooseBlockerOrder(atk, blkA, blkB)
		tg.AssignCombatDamage(atk, map[string]int{
			blkA: 3,
			blkB: 1,
		})
		tg.StopAt(1, core.EndCombat)
		tg.Execute()
		tg.AssertPermanentCount(PlayerA, atk, 0)
		tg.AssertGraveyardCount(PlayerA, atk, 1)
		tg.AssertPermanentCount(PlayerB, blkA, 0)
		tg.AssertGraveyardCount(PlayerB, blkA, 1)
		tg.AssertPermanentCount(PlayerB, blkB, 1)
		tg.AssertGraveyardCount(PlayerB, blkB, 0)
	})

	// CR 510.3a — After combat damage is dealt, triggers that fired during the
	// combat damage step go on the stack and resolve before the active player
	// receives priority in the damage step. So a "whenever ~ deals combat
	// damage to a player, you gain N life" trigger must have resolved before
	// the postcombat main phase begins.
	//
	// Setup: a 3/3 with "Whenever CARDNAME deals combat damage to a player,
	// you gain 2 life." Attack unblocked; defender loses 3, controller gains 2
	// — at StopAt(1, PostcombatMain), PlayerA's life is 22 and PlayerB is 17.
	t.Run("CR 510.3a post-damage trigger resolves before post-damage priority", func(t *testing.T) {
		name := "510.3a Lifegain Attacker 3/3"
		if !mage.CardRegistered(name) {
			mage.Register(name, func() mage.Card {
				// Whenever CARDNAME deals combat damage to a player, you gain 2 life.
				trig := mage.NewTriggered(core.EvtDamageDealt, false,
					mage.GainLife(2)).
					SetCondition(func(evt *core.GameEvent, g mage.GameReader, sourceID, _ uuid.UUID) bool {
						if evt.SourceID != sourceID {
							return false
						}
						if !evt.Flag { // Flag==true for combat damage
							return false
						}
						// TargetID must be a player, not a permanent.
						return g.GetPlayer(evt.TargetID) != nil
					})
				return mage.NewCreature(name, "{2}{W}", 3, 3,
					mage.WithSubTypes("Soldier"),
					mage.WithAbility(trig))
			})
		}
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneBattlefield, PlayerA, name)
		tg.Attack(1, PlayerA, name)
		tg.StopAt(1, core.PostcombatMain)
		tg.Execute()
		tg.AssertLife(PlayerB, 17) // 20 - 3
		tg.AssertLife(PlayerA, 22) // 20 + 2 from the post-damage trigger
	})
}
