package mage

// RequiredSacrificeNumberOrElse reads the count once and requires full payment.
// Each call selects one branch through the shared sacrifice transaction.
func RequiredSacrificeNumberOrElse(player PlayerRef, amount ResolutionValue, filter PermanentFilter, paid, unpaid ResolvingEffect) ResolvingEffect {
	return &requiredSacrificeNumberPayment{player: player, amount: amount, filter: filter, paid: paid, unpaid: unpaid}
}

type requiredSacrificeNumberPayment struct {
	player       PlayerRef
	amount       ResolutionValue
	filter       PermanentFilter
	paid, unpaid ResolvingEffect
}

func (*requiredSacrificeNumberPayment) Text() string { return "mandatory plural sacrifice payment" }
func (e *requiredSacrificeNumberPayment) Properties() EffectProperties {
	var props EffectProperties
	for _, body := range []ResolvingEffect{e.paid, e.unpaid} {
		if !nilDefinition(body) {
			mergeEffectProperties(&props, body.Properties())
		}
	}
	return props
}
func (e *requiredSacrificeNumberPayment) inspectDefinition() definitionNode {
	n := definitionNode{Role: "required-sacrifice-number-payment", Text: e.Text(), children: []definitionEdge{
		{"player", e.player}, {"amount", e.amount},
	}}
	if !nilDefinition(e.paid) {
		n.children = append(n.children, definitionEdge{"paid", e.paid})
	}
	if !nilDefinition(e.unpaid) {
		n.children = append(n.children, definitionEdge{"unpaid", e.unpaid})
	}
	if e.filter.IsZero() {
		n.problem = "supply an explicit sacrifice filter"
	}
	if nilDefinition(e.paid) && nilDefinition(e.unpaid) {
		n.problem = "supply at least one sacrifice payment branch"
	}
	return n
}
func (e *requiredSacrificeNumberPayment) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || resolutionValueReadsSourcePermanent(e.amount) || effectReadsSourcePermanent(e.paid) || effectReadsSourcePermanent(e.unpaid)
}
func (e *requiredSacrificeNumberPayment) Apply(ctx *EffectContext) error {
	id, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	amount := e.amount.resolveValue(ctx)
	paid := amount <= 0
	if amount > 0 {
		cost := SacrificeNMatchingIncludingSourceCost(amount, e.filter, "Sacrifice matching permanents")
		paid = tryResolutionCostPayment(ctx, id, []Cost{cost}, true)
	}
	body := e.unpaid
	if paid {
		body = e.paid
	}
	if nilDefinition(body) {
		return nil
	}
	return body.Apply(ctx)
}
