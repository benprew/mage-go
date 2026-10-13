package mage

import (
	"slices"

	"github.com/google/uuid"
)

// PayAnyOrElse offers eligible players a payment in turn order.
// The first complete payment selects paid. Otherwise, unpaid runs once.
// Either body can be nil. The bodies retain the original effect controller.
func PayAnyOrElse(players PlayerSet, costs CostGroup, paid, unpaid ResolvingEffect) ResolvingEffect {
	return &eligiblePlayerPayment{players: players, costs: costs, paid: paid, unpaid: unpaid}
}

// PayAnyOrElseWithPayer saves each candidate during the offer and payment.
// Only the paid body can read the successful payer. The binding stays local.
// Both bodies retain the original effect controller.
func PayAnyOrElseWithPayer(players PlayerSet, costs CostGroup, payer *PlayerBinding, paid, unpaid ResolvingEffect) ResolvingEffect {
	return &eligiblePlayerPayment{players: players, costs: costs, paid: paid, unpaid: unpaid, payer: payer, scopedPayer: true}
}

type eligiblePlayerPayment struct {
	players      PlayerSet
	costs        CostGroup
	paid, unpaid ResolvingEffect
	payer        *PlayerBinding
	scopedPayer  bool
}

func (*eligiblePlayerPayment) Text() string { return "offer eligible players a payment" }
func (e *eligiblePlayerPayment) Properties() EffectProperties {
	var props EffectProperties
	for _, body := range []ResolvingEffect{e.paid, e.unpaid} {
		if !nilDefinition(body) {
			mergeEffectProperties(&props, body.Properties())
		}
	}
	return props
}
func (e *eligiblePlayerPayment) Apply(ctx *EffectContext) error {
	eligible := e.players.resolvePlayers(ctx)
	if e.scopedPayer {
		if ctx.Game.resolution.playerBindings == nil {
			ctx.Game.resolution.playerBindings = make(map[*PlayerBinding]uuid.UUID)
		}
		previous, present := ctx.Game.resolution.playerBindings[e.payer]
		delete(ctx.Game.resolution.playerBindings, e.payer)
		defer func() {
			delete(ctx.Game.resolution.playerBindings, e.payer)
			if present {
				ctx.Game.resolution.playerBindings[e.payer] = previous
			}
		}()
	}
	body := e.unpaid
	if offerEligibleResolutionCost(ctx, eligible, e.costs.costs, e.payer, "") {
		body = e.paid
	}
	if nilDefinition(body) {
		return nil
	}
	return body.Apply(ctx)
}
func (e *eligiblePlayerPayment) inspectDefinition() definitionNode {
	n := definitionNode{Role: "eligible-player-payment", Text: e.Text(), children: []definitionEdge{{"players", e.players}, {"costs", e.costs}}}
	if e.scopedPayer {
		n.children = append(n.children, definitionEdge{"payer", e.payer})
	}
	if !nilDefinition(e.paid) {
		n.children = append(n.children, definitionEdge{"paid", e.paid})
	}
	if !nilDefinition(e.unpaid) {
		n.children = append(n.children, definitionEdge{"unpaid", e.unpaid})
	}
	if nilDefinition(e.paid) && nilDefinition(e.unpaid) {
		n.problem = "supply at least one payment branch"
	}
	return n
}
func (e *eligiblePlayerPayment) refersToSourcePermanent() bool {
	return playerSetReadsSourcePermanent(e.players) || effectReadsSourcePermanent(e.paid) || effectReadsSourcePermanent(e.unpaid)
}

func offerEligibleResolutionCost(ctx *EffectContext, eligible []uuid.UUID, costs []Cost, payer *PlayerBinding, subject string) bool {
	for _, id := range EachPlayer.resolvePlayers(ctx) {
		if !slices.Contains(eligible, id) {
			continue
		}
		if payer != nil {
			ctx.Game.resolution.playerBindings[payer] = id
		}
		if tryResolutionCostPaymentFor(ctx, id, costs, false, subject) {
			return true
		}
		if payer != nil {
			delete(ctx.Game.resolution.playerBindings, payer)
		}
	}
	return false
}
