package mage

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type permanentControllerRef struct{ subject PermanentRef }
type permanentOwnerRef struct{ subject PermanentRef }

// ControllerOf reads the current controller of the selected battlefield incarnation.
func ControllerOf(subject PermanentRef) PlayerRef { return permanentControllerRef{subject} }
func (permanentControllerRef) playerRef()         {}
func (permanentControllerRef) damageRecipient()   {}
func (r permanentControllerRef) resolvePlayer(ctx *EffectContext) (uuid.UUID, bool) {
	p, ok := r.subject.resolvePermanent(ctx)
	if !ok {
		return uuid.Nil, false
	}
	return p.ControllerID(), ctx.Game.GetPlayer(p.ControllerID()) != nil
}
func (r permanentControllerRef) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return r.resolvePlayer(ctx)
}
func (r permanentControllerRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-ref", Text: "controller", children: []definitionEdge{{"subject", r.subject}}}
}

// OwnerOf reads the owner of the selected battlefield incarnation.
func OwnerOf(subject PermanentRef) PlayerRef { return permanentOwnerRef{subject} }
func (permanentOwnerRef) playerRef()         {}
func (permanentOwnerRef) damageRecipient()   {}
func (r permanentOwnerRef) resolvePlayer(ctx *EffectContext) (uuid.UUID, bool) {
	if nilDefinition(r.subject) {
		return uuid.Nil, false
	}
	p, ok := r.subject.resolvePermanent(ctx)
	if !ok {
		return uuid.Nil, false
	}
	owner := p.Card.Owner()
	return owner, ctx.Game.GetPlayer(owner) != nil
}
func (r permanentOwnerRef) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return r.resolvePlayer(ctx)
}
func (r permanentOwnerRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-ref", Text: "owner", children: []definitionEdge{{"subject", r.subject}}}
}

type unlessPaidEffect struct {
	player PlayerRef
	costs  CostGroup
	effect ResolvingEffect
	ifPaid bool
}

// UnlessPaid offers one player a payment through the total-cost transaction.
// A declined or failed payment runs the effect. An absent player has no effect.
func UnlessPaid(player PlayerRef, costs CostGroup, effect ResolvingEffect) ResolvingEffect {
	return &unlessPaidEffect{player: player, costs: costs, effect: effect}
}

// IfPaid runs the body only after the player accepts and pays the complete cost.
func IfPaid(player PlayerRef, costs CostGroup, effect ResolvingEffect) ResolvingEffect {
	return &unlessPaidEffect{player: player, costs: costs, effect: effect, ifPaid: true}
}
func (e *unlessPaidEffect) Text() string {
	if e.ifPaid {
		return "if the player pays"
	}
	return "unless the player pays"
}
func (e *unlessPaidEffect) Properties() EffectProperties {
	if nilDefinition(e.effect) {
		return EffectProperties{}
	}
	return e.effect.Properties()
}
func (e *unlessPaidEffect) Apply(ctx *EffectContext) error {
	id, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	texts := make([]string, len(e.costs.costs))
	for i, cost := range e.costs.costs {
		texts[i] = cost.Text()
	}
	paid := false
	if ctx.Game.GetPlayer(id).ChooseMayAbility("pay " + strings.Join(texts, ", ")) {
		tx, err := ctx.Game.prepareActionPaymentTransaction(actionPaymentSpec{
			SourceID: ctx.SourceID, Controller: id, Costs: e.costs.costs,
			Targets: ctx.Targets, XValue: ctx.Game.XValue(),
		})
		paid = err == nil && tx.Commit() == nil
	}
	if paid == e.ifPaid {
		return e.effect.Apply(ctx)
	}
	return nil
}
func (e *unlessPaidEffect) inspectDefinition() definitionNode {
	role, branch := "unless-paid", "unpaid"
	if e.ifPaid {
		role, branch = "if-paid", "paid"
	}
	n := definitionNode{Role: role, Text: e.Text(), children: []definitionEdge{{"player", e.player}, {"costs", e.costs}, {branch, e.effect}}}
	return n
}
func (c CostGroup) inspectDefinition() definitionNode {
	n := definitionNode{Role: "cost-group"}
	if !c.present {
		n.problem = "supply an explicit Costs group"
	}
	for i, cost := range c.costs {
		if optional, ok := cost.(*optionalCost); ok && optional.capture {
			n.problem = "optional payment result requires an activated ability cost or spell additional cost"
		}
		if exile, ok := cost.(*exileMatchingCardFromGraveyardCost); ok && exile.capture {
			n.problem = "paid graveyard exile value requires an activated ability cost or spell additional cost"
		}
		if _, ok := cost.(*xLifePayCost); ok {
			n.problem = "paid life value requires an activated ability cost or spell additional cost"
		}
		if life, ok := cost.(*lifePayCost); ok && life.capture {
			n.problem = "paid life value requires an activated ability cost or spell additional cost"
		}
		if _, ok := cost.(*removeAnyCountersCost); ok {
			n.problem = "paid counter value requires an activated ability cost"
		}
		if _, ok := cost.(*capturedXCountersCost); ok {
			n.problem = "paid counter value requires an activated ability cost"
		}
		if counter, ok := cost.(*removeCountersCost); ok && counter.capture {
			n.problem = "paid counter value requires an activated ability cost"
		}
		if sacrifice, ok := cost.(*sacrificeMatchingCost); ok && (sacrifice.captureStats || sacrifice.captureManaValue || sacrifice.captureSubtype) {
			n.problem = "paid sacrifice value requires an activated ability cost or spell additional cost"
		}
		n.children = append(n.children, definitionEdge{fmt.Sprintf("costs[%d]", i), cost})
	}
	return n
}
