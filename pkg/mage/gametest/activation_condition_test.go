package gametest

import (
	"sync"
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

var activationConditionOnce sync.Once

func registerActivationConditionCards() {
	activationConditionOnce.Do(func() {
		// A creature with a tap-for-damage activated ability gated on its
		// CURRENT power being 4 or greater (Bloodshot Trainee pattern). The
		// printed power is 2, so the activation should be rejected until
		// some +N/+0 effect lifts the source above the threshold.
		if !mage.CardRegistered("Test Power Gate Trainee") {
			mage.Register("Test Power Gate Trainee", func() mage.Card {
				return mage.NewCreature("Test Power Gate Trainee", "{2}{R}", 2, 3,
					mage.WithActivatedAbility(
						mage.DealDamage(mage.Fixed(4)),
						mage.TapSourceCost(),
						mage.WithTarget(mage.TargetCreature()),
						mage.WithActivationCondition(func(g *mage.Game, src *mage.Permanent, _ uuid.UUID) bool {
							if src == nil {
								return false
							}
							return src.CurrentPower(g) >= 4
						}),
					),
				)
			})
		}
		// A pump-source spell to push power above the gate.
		if !mage.CardRegistered("Test Pump Source +3/+0") {
			mage.Register("Test Pump Source +3/+0", func() mage.Card {
				return mage.NewInstant("Test Pump Source +3/+0", "{G}",
					mage.NewTargetedSpell(mage.TargetCreature(),
						mage.BoostUntilEndOfTurn(mage.Fixed(3), mage.Fixed(0), mage.SelectTarget)))
			})
		}
		if !mage.CardRegistered("Test Gate Victim") {
			mage.Register("Test Gate Victim", func() mage.Card {
				return mage.NewCreature("Test Gate Victim", "{1}", 1, 1)
			})
		}
	})
}

// TestWithActivationCondition_BlocksWhenFalse verifies that a card-defined
// activation condition rejects the activation while it returns false. The
// trainee's printed power is 2, so the gate (power >= 4) should hold and
// the victim should not take damage.
func TestWithActivationCondition_BlocksWhenFalse(t *testing.T) {
	registerActivationConditionCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Test Power Gate Trainee")
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Test Gate Victim")

	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, "Test Power Gate Trainee", "Test Gate Victim")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	// Activation should have been rejected -> victim still in play, undamaged.
	tg.AssertPermanentCount(PlayerB, "Test Gate Victim", 1)
}

// TestWithActivationCondition_AllowsWhenTrue verifies that the activation is
// permitted once the predicate returns true. Pumping the trainee +3/+0
// (printed 2/3 -> live 5/3) lifts it past the gate; activating the ability
// then deals 4 damage to the 1/1 victim, killing it.
func TestWithActivationCondition_AllowsWhenTrue(t *testing.T) {
	registerActivationConditionCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Test Power Gate Trainee")
	tg.AddCard(core.ZoneHand, PlayerA, "Test Pump Source +3/+0")
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Test Gate Victim")

	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Pump Source +3/+0", "Test Power Gate Trainee")
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, "Test Power Gate Trainee", "Test Gate Victim")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertPermanentCount(PlayerB, "Test Gate Victim", 0)
	tg.AssertGraveyardCount(PlayerB, "Test Gate Victim", 1)
}
