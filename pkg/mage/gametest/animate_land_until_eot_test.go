package gametest

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

// Elemental Uprising shape: target land you control becomes a 4/4
// Elemental creature with haste until end of turn; must be blocked
// this turn if able. This composes existing engine primitives —
// AnimateTargetLand(EndOfTurn) plus TargetMustBeBlockedIfAble — with
// no new engine code. Confirms the XXX marker can be cleared by the
// cards layer using the established API.
func TestAnimateLandUntilEndOfTurn_WithMustBeBlocked(t *testing.T) {
	const cardName = "ALEOT Uprising-like"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			return mage.NewInstant(cardName, "{1}{G}",
				mage.NewTargetedSpell(
					mage.TargetPermanent(mage.IsLand),
					mage.FuncEffect(
						"target land becomes 4/4 Elemental haste; must be blocked",
						mage.EffectProperties{Outcome: mage.OutcomeBenefit},
						func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
							if len(targets) == 0 {
								return nil
							}
							land := g.FindPermanent(targets[0])
							if land == nil {
								return nil
							}
							anim := mage.AnimateTargetLand(land.ID(), mage.AnimateLandOptions{
								Power:     4,
								Toughness: 4,
								SubTypes:  []string{"Elemental"},
								Colors:    []core.Color{core.Green},
								Keywords:  []core.Attr{core.Haste},
							}, core.EndOfTurn)
							anim.SetSourceID(sourceID)
							g.AddContinuousEffect(anim)
							mb := mage.TargetMustBeBlockedIfAble(land.ID(), core.EndOfTurn)
							mb.SetSourceID(sourceID)
							g.AddContinuousEffect(mb)
							return nil
						},
					),
				),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, cardName)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Forest", 5)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, cardName, "Forest")
	tg.StopAt(1, core.PostcombatMain)
	tg.Execute()
	tg.AssertPowerToughness(PlayerA, "Forest", 4, 4)
}
