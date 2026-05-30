package gametest

import (
	"sync"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

var unlessPaysOnce sync.Once

func registerUnlessPaysCards() {
	unlessPaysOnce.Do(func() {
		// "Target player loses 5 life unless they pay 2 life." Modeled
		// purely with LifePayCost so we don't need mana plumbing.
		if !mage.CardRegistered("Test Unless Pays Drain") {
			mage.Register("Test Unless Pays Drain", func() mage.Card {
				return mage.NewSorcery("Test Unless Pays Drain", "{B}",
					mage.NewTargetedSpell(mage.TargetPlayer(),
						mage.UnlessTargetPays(
							mage.SelectTargetPlayer(),
							mage.LifePayCost(2),
							"Pay 2 life to prevent losing 5 life?",
							mage.TargetPlayerLoseLife(mage.Fixed(5)),
						)),
				)
			})
		}
	})
}

// TestUnlessTargetPays_TargetPaysAvoidsEffect verifies that when the
// targeted player accepts and can pay, the inner effect is skipped and
// only the cost is paid (2 life lost from the cost rather than 5 from
// the failed branch).
func TestUnlessTargetPays_TargetPaysAvoidsEffect(t *testing.T) {
	registerUnlessPaysCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Test Unless Pays Drain")
	tg.SetLife(PlayerB, 20)
	// PlayerB defaults to ChooseMayAbility=true → pays the cost.
	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Unless Pays Drain", "PlayerB")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertLife(PlayerB, 18)
}

// TestUnlessTargetPays_TargetDeclinesGetsHit verifies that when the
// targeted player declines, the inner ifNotPaid effect runs (lose 5 life)
// and the cost is not paid.
func TestUnlessTargetPays_TargetDeclinesGetsHit(t *testing.T) {
	registerUnlessPaysCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Test Unless Pays Drain")
	tg.SetLife(PlayerB, 20)

	tpB := tg.GetPlayer(PlayerB)
	tpB.QueueMayAbilityChoices(false)

	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Unless Pays Drain", "PlayerB")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	tg.AssertLife(PlayerB, 15)
}

// TestUnlessTargetPays_CannotPayGetsHit verifies that when the cost is
// genuinely unpayable (target only has 1 life, cost requires 2), the
// ifNotPaid branch runs even though the player would prefer to "pay".
func TestUnlessTargetPays_CannotPayGetsHit(t *testing.T) {
	registerUnlessPaysCards()

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneHand, PlayerA, "Test Unless Pays Drain")
	tg.SetLife(PlayerB, 1)

	tg.CastSpell(1, core.PrecombatMain, PlayerA, "Test Unless Pays Drain", "PlayerB")
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	// Cost (2 life) was unpayable, so the 5-life-loss branch ran. PlayerB
	// drops below 1 life.
	if got := tg.Game.GetPlayer(tg.getPlayerID(PlayerB)).Life(); got > 0 {
		t.Errorf("expected PlayerB life to be <= 0, got %d", got)
	}
}
