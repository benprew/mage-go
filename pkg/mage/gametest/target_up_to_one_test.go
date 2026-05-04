package gametest

import (
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// TargetUpToOneCreature allows resolution with no chosen target.
// (Brightmare: "tap up to one target creature".)
func TestTargetUpToOneCreature_AllowsZeroTargets(t *testing.T) {
	const cardName = "T01 Brightmare-like"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			return mage.NewCreature(cardName, "{1}", 1, 1,
				mage.WithCastTarget(mage.TargetUpToOneCreature()),
				mage.WithETBEffect(mage.FuncEffect("tap optional target",
					mage.EffectProperties{Outcome: mage.OutcomeBenefit},
					func(g *mage.Game, _, _ uuid.UUID, targets []uuid.UUID) error {
						for _, id := range targets {
							if perm := g.FindPermanent(id); perm != nil {
								g.TapPermanent(perm)
							}
						}
						return nil
					})),
			)
		})
	}

	// Cast with NO target available is fine: ETB resolves with empty target list.
	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Plains", 4)
	tg.AddCard(core.ZoneHand, PlayerA, cardName)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, cardName)
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertPermanentCount(PlayerA, cardName, 1)
}

// TargetAnotherCreatureYouControl excludes the source permanent.
// Kira pattern: source's ability targets a creature you control, but never
// itself.
func TestTargetAnotherCreatureYouControl_ExcludesSource(t *testing.T) {
	const cardName = "TARGET-Self-Excluder"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			return mage.NewCreature(cardName, "{1}", 1, 1,
				mage.WithActivatedAbility(
					mage.FuncEffect("tap target another creature you control",
						mage.EffectProperties{},
						func(g *mage.Game, _, _ uuid.UUID, targets []uuid.UUID) error {
							for _, id := range targets {
								if perm := g.FindPermanent(id); perm != nil {
									g.TapPermanent(perm)
								}
							}
							return nil
						}),
					mage.ManaCostOf("{0}"),
					mage.WithTarget(mage.TargetAnotherCreatureYouControl()),
				),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, cardName)
	tg.AddCard(core.ZoneBattlefield, PlayerA, "Grizzly Bears")
	tg.ActivateAbility(2, core.PrecombatMain, PlayerA, cardName, "Grizzly Bears")
	tg.StopAt(2, core.EndStep)
	tg.Execute()

	tg.AssertTapped(PlayerA, "Grizzly Bears", true)
	tg.AssertTapped(PlayerA, cardName, false)
}
