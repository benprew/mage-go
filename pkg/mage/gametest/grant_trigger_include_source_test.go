package gametest

import (
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// GrantTriggeredAbilityToAllIncludingSource lets the source itself receive the
// granted trigger. Kira pattern: "Creatures you control have 'Whenever this
// creature becomes the target of a spell or ability for the first time each
// turn, counter that spell or ability.'"
func TestGrantTriggeredAbilityToAllIncludingSource(t *testing.T) {
	const cardName = "GTAIS Kira-like"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			counter := mage.FuncEffect(
				"counter that spell or ability",
				mage.EffectProperties{Outcome: mage.OutcomeBenefit},
				func(g *mage.Game, _, _ uuid.UUID, targets []uuid.UUID) error {
					if len(targets) < 2 || targets[1] == uuid.Nil {
						return nil
					}
					g.CounterSpellOnStack(targets[1])
					return nil
				},
			)
			return mage.NewCreature(cardName, "{1}{U}{U}", 2, 2,
				mage.WithSubTypes("Spirit"),
				mage.WithStaticAbility(
					mage.GrantTriggeredAbilityToAllIncludingSource(
						core.EvtBecomesTarget, false,
						mage.EventTargetIsSelfFirstTimeThisTurn{},
						mage.IsCreature,
						counter,
					),
				),
			)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, cardName)
	tg.AddCard(core.ZoneBattlefield, PlayerB, "Mountain", 3)
	tg.AddCard(core.ZoneHand, PlayerB, "Lightning Bolt")
	tg.CastSpell(2, core.PrecombatMain, PlayerB, "Lightning Bolt", cardName)
	tg.StopAt(2, core.EndStep)
	tg.Execute()

	// Kira-like is still on the battlefield: the granted trigger countered Bolt.
	tg.AssertPermanentCount(PlayerA, cardName, 1)
	// Bolt is in PlayerB's graveyard (countered = goes to graveyard).
	tg.AssertGraveyardCount(PlayerB, "Lightning Bolt", 1)
}
