package gametest

import (
	"sync"
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

var optionalCostOnce sync.Once

func registerOptionalCostCards() {
	optionalCostOnce.Do(func() {
		// Draconic Roar pattern: instant {1}{R}, deal 3 damage to any target.
		// Additional cost: you may reveal a Dragon card from your hand. If
		// you do, this spell deals an extra 3 damage on top of the base 3.
		dragonCardFilter := mage.NewCardFilter("Dragon card", func(c mage.Card) bool {
			return c.HasSubType("Dragon")
		})
		damageEffect := mage.FuncEffect(
			"deal 3 damage; if reveal cost was paid, deal 3 more",
			mage.EffectProperties{Outcome: mage.OutcomeDetriment},
			func(g *mage.Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
				if len(targets) == 0 {
					return nil
				}
				amt := 3
				if g.LastCostOptionalPaid(sourceID) {
					amt += 3
				}
				tgt := targets[0]
				for _, pl := range g.AllPlayers() {
					if pl.PlayerID() == tgt {
						g.DealDamageToPlayer(pl, amt, sourceID)
						g.ClearOptionalCostPaid(sourceID)
						return nil
					}
				}
				if perm := g.FindPermanent(tgt); perm != nil {
					g.DealDamageToPermanent(perm, amt, sourceID)
				}
				g.ClearOptionalCostPaid(sourceID)
				return nil
			},
		)
		if !mage.CardRegistered("Test Optional Roar") {
			mage.Register("Test Optional Roar", func() mage.Card {
				return mage.NewInstant("Test Optional Roar", "{1}{R}",
					mage.NewTargetedSpell(mage.TargetAnyTarget(), damageEffect),
					mage.WithAdditionalCost(mage.OptionalCost(
						mage.RevealFromHandCost(dragonCardFilter, "Reveal a Dragon card"),
						"Reveal a Dragon card to deal 3 additional damage?",
					)),
				)
			})
		}
		if !mage.CardRegistered("Test Optional Dragon") {
			mage.Register("Test Optional Dragon", func() mage.Card {
				return mage.NewCreature("Test Optional Dragon", "{4}{R}{R}", 5, 5,
					mage.WithSubTypes("Dragon"))
			})
		}
	})
}

// TestOptionalCost_PaidGivesBonus verifies that when the controller
// accepts the optional cost AND has a matching card to reveal, the cost
// is paid and the resolution-time branch reads LastCostOptionalPaid as
// true (extra 3 damage on top of the base 3).
func TestOptionalCost_PaidGivesBonus(t *testing.T) {
	registerOptionalCostCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Test Optional Roar")
	tg.AddCard(core.ZoneHand, PlayerA, "Test Optional Dragon")
	tg.SetLife(PlayerB, 20)
	// TestPlayer.ChooseMayAbility defaults to true → accepts the optional.
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Optional Roar", "PlayerB")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	// Base 3 + bonus 3 = 6 damage to PlayerB.
	tg.AssertLife(PlayerB, 14)
	// Dragon was just revealed, not discarded.
	tg.AssertHandCount(PlayerA, "Test Optional Dragon", 1)
}

// TestOptionalCost_DeclinedNoBonus verifies that when the controller
// declines the optional cost, only the base damage applies.
func TestOptionalCost_DeclinedNoBonus(t *testing.T) {
	registerOptionalCostCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Test Optional Roar")
	tg.AddCard(core.ZoneHand, PlayerA, "Test Optional Dragon")
	tg.SetLife(PlayerB, 20)

	tpA := tg.GetPlayer(PlayerA)
	tpA.QueueMayAbilityChoices(false)

	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Optional Roar", "PlayerB")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	// Only base 3 damage.
	tg.AssertLife(PlayerB, 17)
}

// TestOptionalCost_NoMatchNoBonus verifies that when the controller has
// no matching card in hand to reveal, the optional cost is unpayable and
// the bonus does not apply — but the spell still casts and resolves
// normally for the base 3 damage.
func TestOptionalCost_NoMatchNoBonus(t *testing.T) {
	registerOptionalCostCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Test Optional Roar")
	tg.SetLife(PlayerB, 20)

	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Optional Roar", "PlayerB")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertLife(PlayerB, 17)
}
