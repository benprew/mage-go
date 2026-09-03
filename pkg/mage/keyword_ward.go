package mage

// Ward keyword (CR 702.21).
//
// "Ward [cost]" means: "Whenever this permanent becomes the target of a
// spell or ability an opponent controls, counter that spell or ability
// unless that player pays [cost]."
//
// Implementation details:
//
//   - core.Ward is a keyword attribute. Test functions can check this attribute (CR 702.21a).
//   - WithWard(cost) attaches the display attribute and a triggered ability.
//     The ability triggers on EvtBecomesTarget when an opponent controls the spell or ability.
//   - During resolution, the opponent can pay the ward cost.
//     If the opponent pays the cost, the spell or ability continues.
//     If the opponent does not pay the cost, the engine counters the spell or ability.
//     The function CounterSpellOnStack counters spells and activated abilities.
//     Spells that cannot be countered remain unaffected.
//
// The ability does not trigger when the controller targets their own permanent (CR 702.21b).
// The EventPlayerIsOpponent condition enforces this rule.

import (
	"fmt"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// WithWard adds the Ward keyword (CR 702.21) to a permanent. The cost is
// what an opposing spell or ability's controller must pay to avoid having
// it countered. Pass any Cost — typically ManaCostOf("{2}"), LifePayCost(3),
// or DiscardCost(1).
func WithWard(cost Cost) CardOption {
	return func(c *BaseCard) {
		if c.attrSeeds == nil {
			c.attrSeeds = make(map[Attr]int)
		}
		c.attrSeeds[Ward]++
		c.AddAbility(WardTrigger(cost))
	}
}

// WardTrigger builds the triggered ability for Ward [cost]. Exposed so cards
// that already use WithKeyword(...) for display can still attach the trigger,
// or to allow conditional ward (e.g. modal cards). Most callers should use
// WithWard.
func WardTrigger(cost Cost) *GenericTriggered {
	effect := FuncEffect(
		"counter target spell or ability unless its controller pays the ward cost",
		EffectProperties{Outcome: OutcomeBenefit},
		func(g *Game, sourceID, controller uuid.UUID, targets []uuid.UUID) error {
			// targets[0] is the warded permanent's ID; targets[1] (when present)
			// is the offending spell/ability's source ID. See game.go's
			// EvtBecomesTarget pendingTrigger handling.
			if len(targets) < 2 {
				return nil
			}
			spellSrc := targets[1]
			obj := g.stack.FindBySourceID(spellSrc)
			if obj == nil {
				// The spell/ability already left the stack (resolved, fizzled,
				// or was countered earlier in the queue). Nothing to counter.
				return nil
			}
			payerID := obj.Controller
			payer := g.GetPlayer(payerID)
			if payer == nil {
				return nil
			}
			prompt := fmt.Sprintf("Pay ward cost %s for spell/ability targeting %s?",
				cost.Text(), describeWarded(g, sourceID))
			canPay := cost.CanPay(spellSrc, payerID, g)
			if canPay && payer.ChooseMayAbility(prompt) {
				if err := cost.Pay(spellSrc, payerID, g); err == nil {
					return nil // ward cost paid — spell/ability is not countered
				}
				// Payment failed unexpectedly; fall through to counter.
			}
			g.CounterSpellOnStack(spellSrc)
			return nil
		},
	)
	return NewTriggered(EvtBecomesTarget, false, effect).
		SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
			EventTargetIsSelf{},
			EventPlayerIsOpponent{},
		}})
}

func describeWarded(g *Game, permID uuid.UUID) string {
	if perm := g.FindPermanent(permID); perm != nil {
		return perm.Name()
	}
	return "permanent"
}
