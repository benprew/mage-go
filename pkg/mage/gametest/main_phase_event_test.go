package gametest

import (
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// EvtMainPhase fires once per main phase, with Flag distinguishing precombat
// (true) from postcombat (false). BeginningOfFirstMainPhaseTrigger fires only
// during the controller's precombat main phase. Used by Black Market.
func TestBeginningOfFirstMainPhaseTrigger_FiresOnceOnControllerPrecombatMain(t *testing.T) {
	const cardName = "EVTMAIN First Main Sentinel"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			return mage.NewEnchantment(cardName, "{0}",
				mage.WithAbility(mage.BeginningOfFirstMainPhaseTrigger(
					mage.GainLife(2), false,
				)),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, cardName)
	tg.StopAt(2, core.PrecombatMain)
	tg.Execute()

	// Turn 1 A's first main: +2 (resolves). Turn 1 postcombat: no fire.
	// Turn 1 B's mains: no fire (not controller). Turn 2 A's first main:
	// trigger fires but StopAt halts before priority/resolution, so only
	// the turn-1 +2 life is observable.
	tg.AssertLife(PlayerA, 22)
}

// Postcombat-main trigger fires only on the controller's postcombat main.
func TestBeginningOfPostcombatMainPhaseTrigger_FiresOnControllerPostcombat(t *testing.T) {
	const cardName = "EVTMAIN Postcombat Sentinel"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			return mage.NewEnchantment(cardName, "{0}",
				mage.WithAbility(mage.BeginningOfPostcombatMainPhaseTrigger(
					mage.GainLife(1), false,
				)),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, cardName)
	tg.StopAt(2, core.PrecombatMain)
	tg.Execute()

	// Only A's postcombat main on turn 1 fires (turn 2 stops at PrecombatMain).
	tg.AssertLife(PlayerA, 21)
}

var _ = uuid.UUID{} // keep import stable in case future test wants it
