package gametest

import (
	"sync"
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

var fightOnce sync.Once

func registerFightCards() {
	fightOnce.Do(func() {
		// A spell that makes the SOURCE-attached fighter (designated controller's
		// chosen creature) fight a target. Modeled here as a sorcery whose
		// source is a creature on the battlefield, so we wrap it as an
		// activated ability instead.
		// Simpler: a 4/4 with an activated tap ability "this creature fights
		// target creature you don't control."
		if !mage.CardRegistered("Test Fight Ox") {
			mage.Register("Test Fight Ox", func() mage.Card {
				return mage.NewCreature("Test Fight Ox", "{2}{G}", 4, 4,
					mage.WithActivatedAbility(
						mage.FightTarget(),
						mage.TapSourceCost(),
						mage.WithTarget(mage.TargetCreature()),
					))
			})
		}
		if !mage.CardRegistered("Test Fight 1/1") {
			mage.Register("Test Fight 1/1", func() mage.Card {
				return mage.NewCreature("Test Fight 1/1", "{1}", 1, 1)
			})
		}
		if !mage.CardRegistered("Test Fight 5/5") {
			mage.Register("Test Fight 5/5", func() mage.Card {
				return mage.NewCreature("Test Fight 5/5", "{4}", 5, 5)
			})
		}
		// A creature with a "when this dies, gain 3 life" delayed-trigger
		// activated ability: tap, register OnPermanentDiesThisTurn(self).
		if !mage.CardRegistered("Test On-Death Self Watcher") {
			mage.Register("Test On-Death Self Watcher", func() mage.Card {
				return mage.NewCreature("Test On-Death Self Watcher", "{1}{W}", 1, 2,
					mage.WithActivatedAbility(
						mage.OnTargetDiesThisTurn(mage.GainLife(3)),
						mage.TapSourceCost(),
						mage.WithTarget(mage.TargetCreature()),
					))
			})
		}
	})
}

// TestFightTarget_KillsBoth verifies CR 701.13: source and target each
// deal damage equal to their power to the other simultaneously. A 4/4
// fighting a 5/5 leaves the 5/5 with 4 damage (lethal because 4 >= 5 is
// false; it survives with 4 damage marked) and the 4/4 with 5 damage
// (lethal). State-based actions clean up the dead 4/4.
func TestFightTarget_KillsAttacker(t *testing.T) {
	registerFightCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Test Fight Ox")
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Test Fight 5/5")

	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, "Test Fight Ox", "Test Fight 5/5")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, "Test Fight Ox", 0)
	tg.AssertGraveyardCount(PlayerA, "Test Fight Ox", 1)
	tg.AssertPermanentCount(PlayerB, "Test Fight 5/5", 1)
}

// TestFightTarget_KillsBoth: a 4/4 fights a 4/4-equivalent (modeled as
// the 5/5 with toughness lowered? simpler — make a 4/4 victim by using
// the 1/1 and asserting only the 1/1 dies).
func TestFightTarget_KillsTarget(t *testing.T) {
	registerFightCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Test Fight Ox")
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Test Fight 1/1")

	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, "Test Fight Ox", "Test Fight 1/1")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertGraveyardCount(PlayerB, "Test Fight 1/1", 1)
	tg.AssertPermanentCount(PlayerA, "Test Fight Ox", 1)
}

// TestOnPermanentDiesThisTurn_Fires verifies the delayed-trigger primitive:
// after activating "when target dies this turn, gain 3 life" on a 1/1, if
// the 1/1 then dies before end of turn (e.g. by taking lethal damage), the
// trigger fires and the activator gains 3 life.
func TestOnPermanentDiesThisTurn_Fires(t *testing.T) {
	registerFightCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Test On-Death Self Watcher")
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Test Fight Ox")
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Test Fight 1/1")
	tg.SetLife(PlayerA, 20)

	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, "Test On-Death Self Watcher", "Test Fight 1/1")
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, "Test Fight Ox", "Test Fight 1/1")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertGraveyardCount(PlayerB, "Test Fight 1/1", 1)
	tg.AssertLife(PlayerA, 23)
}

// TestOnPermanentDiesThisTurn_DoesNotFireIfAlive: if the watched creature
// is still alive at end of turn, the trigger never fires and the delayed
// trigger is cleaned up at end-of-turn (no life gain).
func TestOnPermanentDiesThisTurn_DoesNotFireIfAlive(t *testing.T) {
	registerFightCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Test On-Death Self Watcher")
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Test Fight 5/5")
	tg.SetLife(PlayerA, 20)

	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, "Test On-Death Self Watcher", "Test Fight 5/5")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertLife(PlayerA, 20)
}
