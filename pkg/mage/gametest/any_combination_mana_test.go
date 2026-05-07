package gametest

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// AnyCombination=true on a multi-amount ManaProduction lets the controller
// pick a different color for each mana point. Selvala-style "Add X mana in
// any combination of colors".
func TestManaProduction_AnyCombination_PerPointChoice(t *testing.T) {
	const cardName = "ANYCOMB Selvala-like"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			return mage.NewCreature(cardName, "{1}{G}{G}", 2, 3,
				mage.WithSubTypes("Elf", "Scout"),
				mage.WithMultiManaAbility(mage.ManaProduction{
					Color: core.AnyColor, Amount: 3, AnyCombination: true,
				}),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, cardName)
	tg.ActivateAbility(2, core.PrecombatMain, PlayerA, cardName)
	tg.ChooseManaColor(PlayerA, core.Red)
	tg.ChooseManaColor(PlayerA, core.Blue)
	tg.ChooseManaColor(PlayerA, core.Green)
	tg.StopAt(2, core.EndStep)
	tg.Execute()

	tg.AssertManaProducedAtLeast(PlayerA, core.Red, 1)
	tg.AssertManaProducedAtLeast(PlayerA, core.Blue, 1)
	tg.AssertManaProducedAtLeast(PlayerA, core.Green, 1)
}
