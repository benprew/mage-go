package mage

// Ward keyword (CR 702.21).
//
// "Ward [cost]" means: "Whenever this permanent becomes the target of a
// spell or ability an opponent controls, counter that spell or ability
// unless that player pays [cost]."
//
// Implementation:
//
//   - core.Ward is a keyword Attr so AssertHasAbility(player, name, core.Ward,
//     true) works for display/test purposes (CR 702.21a — ward is a
//     triggered ability, not a static ability, but it still appears in the
//     creature's keyword list).
//   - WithWard(cost) wires both the display Attr and a triggered ability
//     that fires on EvtBecomesTarget when the spell/ability's controller
//     is an opponent of the warded permanent's controller.
//   - On resolution, the opponent is offered a may-pay choice. If they pay
//     (or can pay and choose to), the spell/ability proceeds. If they
//     decline or can't pay, the spell/ability is countered via
//     CounterSpellOnStack (which also covers activated abilities — both
//     spells and abilities are removed by their stack-object source ID
//     match). For triggered abilities — which can't be the target of
//     ward according to CR 702.21a's "spell or ability an opponent
//     controls" wording, since triggered abilities also count — we still
//     try to counter; uncounterable spells (WithUncounterable / static
//     filters) are unaffected per CounterSpellOnStack.
//
// Ward triggers do NOT trigger when the warded permanent itself is
// targeted by its own controller (CR 702.21b — only opponents). This is
// enforced by the EventPlayerIsOpponent condition.

import (
	"fmt"

	"github.com/google/uuid"

	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
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
