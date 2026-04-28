package gametest

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// RevealTopAndDealDamage (Riddle of Lightning shape): the controller
// reveals the top card of their library and the spell deals damage
// equal to that card's mana value to the chosen target. The library is
// not mutated by the reveal.
func TestRevealTopAndDealDamage(t *testing.T) {
	const cardName = "RTAD Riddle-like"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			return mage.NewInstant(cardName, "{3}{R}{R}",
				mage.NewTargetedSpell(mage.TargetAnyTarget(), mage.RevealTopAndDealDamage()),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, cardName)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Mountain", 5)
	// Top of library is Hill Giant (CMC 4).
	tg.AddCard(core.ZoneLibrary, PlayerA, "Hill Giant")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Mountain", 5)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, cardName, "PlayerB")
	tg.StopAt(1, core.EndStep)
	tg.Execute()
	tg.AssertLife(PlayerB, 16)
	tg.AssertLibraryTop(PlayerA, "Hill Giant")
}
