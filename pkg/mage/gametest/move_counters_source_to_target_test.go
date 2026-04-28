package gametest

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// MoveCountersFromSourceToTarget primitive (Scrounging Bandar / Power
// Conduit pattern). The controller is prompted via ChooseNumber(0,
// available, reason) and the chosen count of counters is removed from
// the source permanent and added to the target permanent.
func TestMoveCountersFromSourceToTarget(t *testing.T) {
	const cardName = "MCST Conduit-like"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			return mage.NewCreature(cardName, "{1}{G}", 2, 2,
				mage.WithSubTypes("Cat", "Monkey"),
				mage.WithActivatedAbility(
					mage.MoveCountersFromSourceToTarget(core.P1P1),
					mage.ManaCostOf("{0}"),
					mage.WithTarget(mage.TargetAnotherCreatureYouControl()),
				),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, cardName)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	tg.AddCounters(1, core.PrecombatMain, PlayerA, cardName, core.P1P1, 2)
	tg.ChooseNumber(PlayerA, 2)
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, cardName, "Grizzly Bears")
	tg.StopAt(1, core.EndStep)
	tg.Execute()
	tg.AssertCounterCount(PlayerA, cardName, core.P1P1, 0)
	tg.AssertCounterCount(PlayerA, "Grizzly Bears", core.P1P1, 2)
	tg.AssertPowerToughness(PlayerA, "Grizzly Bears", 4, 4)
}
