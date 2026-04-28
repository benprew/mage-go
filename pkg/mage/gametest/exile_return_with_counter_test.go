package gametest

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// ExileTargetReturnAtEndStepWithCounter (Long Road Home pattern): exile
// target creature; at the beginning of the next end step, return that
// card to the battlefield under its owner's control with a +1/+1
// counter on it.
func TestExileTargetReturnAtEndStepWithCounter(t *testing.T) {
	const cardName = "ERC Long Road Home-like"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			return mage.NewInstant(cardName, "{1}{W}",
				mage.NewTargetedSpell(
					mage.TargetCreature(),
					mage.ExileTargetReturnAtEndStepWithCounter(core.P1P1, 1),
				),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, cardName)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Plains", 2)
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Grizzly Bears")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, cardName, "Grizzly Bears")
	tg.StopAt(2, core.Upkeep)
	tg.Execute()
	// After EOT of turn 1, Grizzly Bears returns with +1/+1.
	tg.AssertPermanentCount(PlayerB, "Grizzly Bears", 1)
	tg.AssertPowerToughness(PlayerB, "Grizzly Bears", 3, 3)
}
