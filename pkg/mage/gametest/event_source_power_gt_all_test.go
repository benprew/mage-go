package gametest

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// EventSourcePowerGreaterThanAllOthers gates a "whenever another creature
// enters" trigger on the entering creature having strictly greater power
// than every other creature on the battlefield. Selvala, Heart of the
// Wilds shape: the trigger fires when a 4-power creature enters but not
// when a 2-power creature enters into a board with a 3-power creature
// already present.
func TestEventSourcePowerGreaterThanAllOthers(t *testing.T) {
	const cardName = "ESPGTAO Selvala-like"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			trig := mage.WheneverPermanentEntersBattlefieldTrigger(
				mage.DrawCards(mage.Fixed(1)), false, mage.IsCreature,
			).AndConditionData(mage.EventSourceNotSelf{}).
				AndConditionData(mage.EventSourcePowerGreaterThanAllOthers{})
			return mage.NewCreature(cardName, "{1}{G}{G}", 2, 3,
				mage.WithSubTypes("Elf", "Scout"),
				mage.WithAbility(trig),
			)
		})
	}

	// Board: Selvala (2/3) for PlayerA. Cast Grizzly Bears (2/2): not greater
	// than Selvala's 2 — no draw. Then play Hill Giant (3/3): greater than
	// 2 and 2 — draw a card.
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, cardName)
	tg.AddCard(core.ZoneHand, PlayerA, "Grizzly Bears")
	tg.AddCard(core.ZoneHand, PlayerA, "Hill Giant")
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Forest", 6)
	tg.AddCard(core.ZoneLibrary, PlayerA, "Mountain", 5)
	tg.CastSpell(3, core.PrecombatMain, PlayerA, "Grizzly Bears")
	tg.CastSpell(3, core.PrecombatMain, PlayerA, "Hill Giant")
	tg.StopAt(3, core.EndStep)
	tg.Execute()
	tg.AssertHandCount(PlayerA, "Mountain", 1)
}
