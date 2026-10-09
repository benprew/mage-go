package mage

type resolutionLifePayment struct {
	player PlayerRef
	amount ResolutionValue
	effect ResolvingEffect
	ifPaid bool
}

// UnlessLifePaid reads the amount once before an optional life payment.
// A declined or failed payment runs the effect.
func UnlessLifePaid(player PlayerRef, amount ResolutionValue, effect ResolvingEffect) ResolvingEffect {
	return &resolutionLifePayment{player: player, amount: amount, effect: effect}
}

// IfLifePaid runs the effect after the player accepts and pays the life cost.
func IfLifePaid(player PlayerRef, amount ResolutionValue, effect ResolvingEffect) ResolvingEffect {
	return &resolutionLifePayment{player: player, amount: amount, effect: effect, ifPaid: true}
}

func (*resolutionLifePayment) Text() string { return "optional resolution life payment" }
func (e *resolutionLifePayment) Properties() EffectProperties {
	if nilDefinition(e.effect) {
		return EffectProperties{}
	}
	return e.effect.Properties()
}
func (e *resolutionLifePayment) Apply(ctx *EffectContext) error {
	amount := max(0, e.amount.resolveValue(ctx))
	return (&unlessPaidEffect{player: e.player, costs: Costs(LifePayCost(amount)), effect: e.effect, ifPaid: e.ifPaid}).Apply(ctx)
}
func (e *resolutionLifePayment) inspectDefinition() definitionNode {
	role, branch := "unless-paid", "unpaid"
	if e.ifPaid {
		role, branch = "if-paid", "paid"
	}
	return definitionNode{Role: role, Text: e.Text(), children: []definitionEdge{{"player", e.player}, {"amount", e.amount}, {branch, e.effect}}}
}
func (e *resolutionLifePayment) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || resolutionValueReadsSourcePermanent(e.amount) || effectReadsSourcePermanent(e.effect)
}
