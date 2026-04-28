package gametest

import (
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// AddLifeGainModifier (Rhox Faithmender pattern): "If you would gain
// life, you gain twice that much life instead." Registered as a static
// modifier on a permanent; auto-cleared when source leaves.
func TestAddLifeGainModifier_DoubleLifegain(t *testing.T) {
	const cardName = "LGM Faithmender-like"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			return mage.NewCreature(cardName, "{3}{W}", 1, 5,
				mage.WithSubTypes("Rhino", "Monk"),
				mage.WithAbility(
					mage.EntersBattlefieldTrigger(
						mage.FuncEffect("install life-gain doubler",
							mage.EffectProperties{Outcome: mage.OutcomeBenefit},
							func(g *mage.Game, sourceID, controller uuid.UUID, _ []uuid.UUID) error {
								g.AddLifeGainModifier(sourceID, func(g *mage.Game, gainingID uuid.UUID, amount int) int {
									if gainingID == controller {
										return amount * 2
									}
									return amount
								})
								return nil
							},
						),
						false,
					),
				),
			)
		})
	}

	const triggerName = "LGM Heal3"
	if !mage.CardRegistered(triggerName) {
		mage.Register(triggerName, func() mage.Card {
			return mage.NewSorcery(triggerName, "{W}",
				mage.NewSpellAbility(mage.GainLife(3)),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, cardName)
	tg.AddCard(core.ZoneHand, PlayerA, triggerName)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Plains", 5)
	tg.SetLife(PlayerA, 20)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, cardName)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, triggerName)
	tg.StopAt(1, core.EndStep)
	tg.Execute()
	tg.AssertLife(PlayerA, 26) // 20 + (3 * 2)
}
