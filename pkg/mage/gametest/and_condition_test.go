package gametest

import (
	"testing"

	"github.com/google/uuid"

	"git.sr.ht/~cdcarter/mage-go/pkg/mage"
	"git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
)

// AndConditionData composes a refining predicate with any condition installed
// by a constructor (e.g. the filter passed to
// WheneverPermanentEntersBattlefieldTrigger). Without composition, chaining
// SetConditionData would replace the constructor's filter — causing
// nontoken-ETB triggers like Lathliss to fire on the very token they create.
func TestAndConditionData_PreservesConstructorFilter(t *testing.T) {
	const cardName = "ANDCOND Token Maker"
	if !mage.CardRegistered(cardName) {
		mage.Register(cardName, func() mage.Card {
			nontokenCreature := mage.NewPermanentFilter("nontoken creature", func(p *mage.Permanent, _ *mage.Game) bool {
				return p.HasType(core.TypeCreature) && !p.Card.IsToken()
			})
			return mage.NewEnchantment(cardName, "{0}",
				mage.WithAbility(
					mage.WheneverPermanentEntersBattlefieldTrigger(
						mage.CreateToken("Beast", 2, 2, []core.CardType{core.TypeCreature}, []string{"Beast"}),
						false,
						nontokenCreature,
					).AndConditionData(mage.AndTriggerCond{Conditions: []mage.TriggerConditionData{
						mage.EventSourceNotSelf{},
						mage.EventSourceControlledByController{},
					}}),
				),
			)
		})
	}

	const fakeDragonName = "ANDCOND Beast Maker"
	if !mage.CardRegistered(fakeDragonName) {
		mage.Register(fakeDragonName, func() mage.Card {
			return mage.NewCreature(fakeDragonName, "{0}", 1, 1)
		})
	}

	tg := NewTestGame(t)
	tg.AddCard(core.ZoneBattlefield, PlayerA, cardName)
	tg.AddCard(core.ZoneHand, PlayerA, fakeDragonName)
	tg.CastSpell(1, core.PrecombatMain, PlayerA, fakeDragonName)
	tg.StopAt(1, core.EndStep)
	tg.Execute()

	// Exactly ONE Beast token — the trigger fires once for the nontoken
	// creature ETB, and does NOT re-fire on the token's own ETB because
	// the constructor filter (nontoken) is preserved.
	tg.AssertPermanentCount(PlayerA, "Beast", 1)
}

// Regression: SetConditionData still REPLACES (existing behavior preserved
// for back-compat). AndConditionData is the additive variant.
func TestSetConditionData_StillReplaces(t *testing.T) {
	called := false
	mage.AsTriggerCondition(mage.EventSourceIsSelf{}) // touch import
	cond := func(_ *core.GameEvent, _ mage.GameReader, _, _ uuid.UUID) bool {
		called = true
		return false
	}
	trig := mage.NewTriggered(core.EvtEntersBattlefield, false).SetCondition(cond)
	// Replace.
	trig.SetConditionData(mage.EventSourceIsSelf{})
	// Original closure no longer invoked.
	_ = trig.CheckTrigger(&core.GameEvent{}, nil)
	if called {
		t.Fatalf("SetConditionData should replace, not compose")
	}
}
