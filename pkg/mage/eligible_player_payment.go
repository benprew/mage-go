package mage

import "slices"

// PayAnyOrElse offers eligible players a payment in turn order.
// The first complete payment selects paid. Otherwise, unpaid runs once.
// Either body can be nil. The bodies retain the original effect controller.
func PayAnyOrElse(players PlayerSet, costs CostGroup, paid, unpaid ResolvingEffect) ResolvingEffect {
	return &eligiblePlayerPayment{players, costs, paid, unpaid}
}

type eligiblePlayerPayment struct {
	players      PlayerSet
	costs        CostGroup
	paid, unpaid ResolvingEffect
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
	body := e.unpaid
	for _, id := range EachPlayer.resolvePlayers(ctx) {
		if slices.Contains(eligible, id) && tryResolutionCostPayment(ctx, id, e.costs.costs, false) {
			body = e.paid
			break
		}
	}
	if nilDefinition(body) {
		return nil
	}
	return body.Apply(ctx)
}
func (e *eligiblePlayerPayment) inspectDefinition() definitionNode {
	n := definitionNode{Role: "eligible-player-payment", Text: e.Text(), children: []definitionEdge{{"players", e.players}, {"costs", e.costs}}}
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
