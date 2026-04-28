package gametest

import (
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// SacrificeSelfTrigger fires when the source is sacrificed, even though
// Game.Sacrifice removes the permanent from the battlefield before firing
// the LtB event. Terrarion pattern.
func TestSacrificeSelfTrigger_FiresOnSacrifice(t *testing.T) {
	const cardName = "SST Terrarion-like"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			return mage.NewArtifact(cardName, "{1}",
				mage.WithActivatedAbility(
					mage.FuncEffect("nothing", mage.EffectProperties{},
						func(g *mage.Game, _, _ uuid.UUID, _ []uuid.UUID) error { return nil }),
					mage.ManaCostOf("{0}"),
					mage.WithCost(mage.SacrificeSourceCost()),
				),
				mage.WithAbility(mage.LeavesBattlefieldToGraveyardTrigger(
					mage.DrawCards(mage.Fixed(1)), false,
				)),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, cardName)
	tg.AddCard(core.ZoneLibrary, PlayerA, "Mountain", 5)
	tg.ActivateAbility(1, core.PrecombatMain, PlayerA, cardName)
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, cardName, 0)
	tg.AssertGraveyardCount(PlayerA, cardName, 1)
	// Trigger drew a Mountain. The default test player auto-plays lands, so
	// the drawn Mountain ends up on the battlefield by end step.
	tg.AssertPermanentCount(PlayerA, "Mountain", 1)
}
