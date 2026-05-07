package gametest

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// Fell Specter pattern: "When this creature enters, target opponent
// discards a card." This confirms the existing
// EntersBattlefieldTrigger + AddTarget(TargetOpponent()) +
// DiscardCards primitives compose without engine-side gaps.
func TestETB_TargetOpponentDiscards(t *testing.T) {
	const cardName = "ETB Target Opponent Discards"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			return mage.NewCreature(cardName, "{3}{B}", 1, 3,
				mage.WithSubTypes("Specter"),
				mage.WithAbility(
					mage.EntersBattlefieldTrigger(
						mage.DiscardCards(mage.Fixed(1)),
						false,
					).AddTarget(mage.TargetOpponent()),
				),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, cardName)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Swamp", 4)
	tg.AddCard(core.ZoneHand, PlayerB, "Mountain", 3)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, cardName)
	tg.ChooseDiscard(PlayerB, "Mountain")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertHandCount(PlayerB, "Mountain", 2)
	tg.AssertGraveyardCount(PlayerB, "Mountain", 1)
}
