package gametest

import (
	"sync"
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

var eventSourceIDOnce sync.Once

// capturedDamageSource tracks the SourceID exposed during the trigger
// resolution of "Test Vengeful Bear". It is set to the damager's UUID
// when the trigger fires.
var capturedDamageSource uuid.UUID

func registerEventSourceIDCards() {
	eventSourceIDOnce.Do(func() {
		// "Whenever this creature is dealt damage, do something with the
		// damaging source." We capture g.EventSourceID() during resolution.
		if !mage.CardRegistered("Test Vengeful Bear") {
			mage.Register("Test Vengeful Bear", func() mage.Card {
				captureEffect := mage.FuncEffect(
					"capture damage source",
					mage.EffectProperties{},
					func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
						capturedDamageSource = g.EventSourceID()
						return nil
					},
				)
				return mage.NewCreature("Test Vengeful Bear", "{1}{G}", 4, 4,
					mage.WithSubTypes("Bear"),
					mage.WithAbility(mage.WhenDamageDealtToThisTrigger(captureEffect, false)),
				)
			})
		}
		if !mage.CardRegistered("Test ESID Bolt") {
			mage.Register("Test ESID Bolt", func() mage.Card {
				return mage.NewSorcery("Test ESID Bolt", "{R}",
					mage.NewTargetedSpell(mage.TargetCreature(), mage.DealDamage(mage.Fixed(2))),
				)
			})
		}
	})
}

// TestEventSourceID_OnDamageTrigger verifies that during the resolution
// of an EvtDamageDealt-triggered ability, Game.EventSourceID() returns
// the damager's permanent (or spell) ID, allowing FuncEffect-based
// resolution to identify the damage source.
func TestEventSourceID_OnDamageTrigger(t *testing.T) {
	registerEventSourceIDCards()

	tg := NewTestGame(t)
	bearID := tg.AddCard(core.ZoneBattlefield, PlayerA, "Test Vengeful Bear")
	tg.AddCard(core.ZoneHand, PlayerB, "Test ESID Bolt")

	capturedDamageSource = uuid.Nil

	// PlayerB casts Test ESID Bolt at the bear, dealing 2 damage.
	tg.CastSpell(2, core.PrecombatMain, PlayerB, "Test ESID Bolt", "Test Vengeful Bear")
	tg.StopAt(2, core.EndStep)
	tg.Execute()

	if capturedDamageSource == uuid.Nil {
		t.Fatal("trigger did not fire (or EventSourceID was nil)")
	}
	if capturedDamageSource == bearID {
		t.Errorf("EventSourceID returned the bear (the damaged), want the damaging source")
	}
	// The captured ID should be the bolt spell's source — a card not on the battlefield
	// at resolution time. We just verify it's distinct from the bear and non-nil.
}
