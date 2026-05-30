package gametest

import (
	"sync"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

var eitherCostOnce sync.Once

func registerEitherCostCards() {
	eitherCostOnce.Do(func() {
		// "As an additional cost, discard a card or pay {5}."
		// Plus a small body so we can detect the cast resolved.
		if !mage.CardRegistered("Test Branching Cost Spell") {
			mage.Register("Test Branching Cost Spell", func() mage.Card {
				return mage.NewSorcery("Test Branching Cost Spell", "{2}{B}",
					mage.NewSpellAbility(mage.GainLife(2)),
					mage.WithAdditionalCost(mage.EitherCost(
						mage.DiscardCost(1),
						mage.ManaCostOf("{5}"),
					)),
				)
			})
		}
	})
}

// TestEitherCost_PicksFirstByDefault confirms the EitherCost selects the
// first payable branch when both are payable (BasePlayer.ChooseMode returns
// 0 by default — discard). The spell still resolves.
func TestEitherCost_PicksFirstByDefault(t *testing.T) {
	registerEitherCostCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Plains")
	tg.AddCard(core.ZoneHand, PlayerA, "Test Branching Cost Spell")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Branching Cost Spell")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	// Spell resolved, gained 2 life.
	tg.AssertLife(PlayerA, 22)
	// Plains was discarded as the additional cost.
	tg.AssertGraveyardCount(PlayerA, "Plains", 1)
	tg.AssertHandCount(PlayerA, "Plains", 0)
}

// TestEitherCost_FallsBackToOnlyPayable confirms when only one branch can
// be paid, that branch is selected automatically without a player prompt.
// (Uncastability when no branch is payable is covered by the cost subsystem.)
func TestEitherCost_FallsBackToOnlyPayable(t *testing.T) {
	registerEitherCostCards()

	tg := NewTestGame(t)
	// Hand has only the spell itself — it'll be on the stack during cost
	// payment, so discard has no card to take. Only "{5}" is payable.
	tg.AddCard(core.ZoneHand, PlayerA, "Test Branching Cost Spell")
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Branching Cost Spell")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertLife(PlayerA, 22)
}
