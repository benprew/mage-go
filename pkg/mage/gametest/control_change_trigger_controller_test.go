package gametest

import (
	"sync"
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

var controlChangeTriggerControllerOnce sync.Once

func registerControlChangeTriggerControllerCards() {
	controlChangeTriggerControllerOnce.Do(func() {
		mage.Register("Trigger Controller Upkeep Creature", func() mage.Card {
			return mage.NewCreature("Trigger Controller Upkeep Creature", "{3}{U}", 4, 1,
				mage.WithAbility(mage.SacrificeAtUpkeepUnlessPay("{U}")),
			)
		})
		mage.Register("Trigger Controller Control Aura", func() mage.Card {
			return mage.NewAura("Trigger Controller Control Aura", "{2}{U}{U}",
				mage.WithStaticAbility(mage.ControlChangeContinuous()),
			)
		})
	})
}

func TestControlChangeUpdatesSourceTriggeredAbilityController(t *testing.T) {
	registerControlChangeTriggerControllerCards()

	t.Run("stolen source does not trigger on previous controller upkeep", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneHand, PlayerA, "Trigger Controller Upkeep Creature")
		tg.AddCard(core.ZoneHand, PlayerB, "Trigger Controller Control Aura")

		tg.CastSpell(1, core.PrecombatMain, PlayerA, "Trigger Controller Upkeep Creature")
		tg.CastSpell(2, core.PrecombatMain, PlayerB, "Trigger Controller Control Aura", "Trigger Controller Upkeep Creature")
		tg.StopAt(3, core.PrecombatMain)
		tg.Execute()

		tg.AssertPermanentCount(PlayerA, "Trigger Controller Upkeep Creature", 0)
		tg.AssertPermanentCount(PlayerB, "Trigger Controller Upkeep Creature", 1)
		tg.AssertGraveyardCount(PlayerA, "Trigger Controller Upkeep Creature", 0)
	})

	t.Run("stolen source triggers on new controller upkeep", func(t *testing.T) {
		tg := NewTestGame(t)
		tg.AddCard(core.ZoneHand, PlayerA, "Trigger Controller Upkeep Creature")
		tg.AddCard(core.ZoneHand, PlayerB, "Trigger Controller Control Aura")

		tg.CastSpell(1, core.PrecombatMain, PlayerA, "Trigger Controller Upkeep Creature")
		tg.CastSpell(2, core.PrecombatMain, PlayerB, "Trigger Controller Control Aura", "Trigger Controller Upkeep Creature")
		tg.StopAt(4, core.PrecombatMain)
		tg.Execute()

		tg.AssertPermanentCount(PlayerA, "Trigger Controller Upkeep Creature", 0)
		tg.AssertPermanentCount(PlayerB, "Trigger Controller Upkeep Creature", 0)
		tg.AssertGraveyardCount(PlayerA, "Trigger Controller Upkeep Creature", 1)
	})
}
