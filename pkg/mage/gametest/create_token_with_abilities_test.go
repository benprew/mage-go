package gametest

import (
	"testing"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// CreateTokensWithAbilities (Dance with Devils pattern): "Create two
// 1/1 red Devil creature tokens. They have 'When this token dies, it
// deals 1 damage to any target.'" The created tokens carry their own
// triggered ability — when one dies, it fires.
func TestCreateTokensWithAbilities_DanceWithDevils(t *testing.T) {
	const cardName = "CTWA Dance-with-Devils-like"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			tokenAbility := mage.PutIntoGraveyardFromBattlefieldTrigger(
				mage.DealDamage(mage.Fixed(1)), false,
			).AddTarget(mage.TargetPlayer())
			return mage.NewInstant(cardName, "{3}{R}",
				mage.NewSpellAbility(mage.CreateTokensWithAbilities(
					2, "Devil", 1, 1,
					[]core.CardType{core.TypeCreature},
					[]string{"Devil"},
					nil,
					tokenAbility,
				)),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, cardName)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Mountain", 4)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, cardName)
	tg.StopAt(1, core.PostcombatMain)
	tg.Execute()
	tg.AssertPermanentCount(PlayerA, "Devil", 2)
}
