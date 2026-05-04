package gametest

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// EvtCombatDamageDealt fires once per (controller, recipient-player) pair per
// combat damage step. Keeper of Fables pattern: "Whenever one or more
// creatures you control deal combat damage to a player, draw a card."
// Two creatures attack and connect — exactly one card should be drawn.
func TestWheneverOneOrMoreCreaturesYouControlDealCombatDamageToPlayer_DrawsOnce(t *testing.T) {
	const cardName = "CDD Keeper-like"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			return mage.NewCreature(cardName, "{3}{G}{G}", 4, 5,
				mage.WithSubTypes("Cat"),
				mage.WithAbility(mage.WheneverOneOrMoreCreaturesYouControlDealCombatDamageToPlayerTrigger(
					mage.DrawCards(mage.Fixed(1)), false,
				)),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, cardName)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	tg.AddCard(core.ZoneLibrary, PlayerA, "Mountain", 5)
	tg.Attack(3, PlayerA, cardName, "Grizzly Bears")
	tg.StopAt(3, core.EndStep)
	tg.Execute()

	// Both attackers connected: PlayerB took combat damage from two creatures
	// controlled by PlayerA, but the trigger fires once.
	tg.AssertHandCount(PlayerA, "Mountain", 1)
	// PlayerB still alive, took 4+2 = 6 damage.
	tg.AssertLife(PlayerB, 14)
}
