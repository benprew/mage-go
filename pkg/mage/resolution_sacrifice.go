package mage

// IfSacrificePaid runs the body after the player accepts and sacrifices the subject.
func IfSacrificePaid(player PlayerRef, subject PermanentRef, body ResolvingEffect) ResolvingEffect {
	return &sacrificePaymentEffect{player: player, subject: subject, body: body, ifPaid: true}
}

// UnlessSacrificePaid runs the body when the player declines or cannot sacrifice the subject.
func UnlessSacrificePaid(player PlayerRef, subject PermanentRef, body ResolvingEffect) ResolvingEffect {
	return &sacrificePaymentEffect{player: player, subject: subject, body: body}
}

// IfRequiredSacrificePaid runs the body after mandatory payment of the subject.
func IfRequiredSacrificePaid(player PlayerRef, subject PermanentRef, body ResolvingEffect) ResolvingEffect {
	return &sacrificePaymentEffect{player: player, subject: subject, body: body, ifPaid: true, required: true}
}

// UnlessRequiredSacrificePaid runs the body when mandatory payment is unavailable.
func UnlessRequiredSacrificePaid(player PlayerRef, subject PermanentRef, body ResolvingEffect) ResolvingEffect {
	return &sacrificePaymentEffect{player: player, subject: subject, body: body, required: true}
}

type sacrificePaymentEffect struct {
	player   PlayerRef
	subject  PermanentRef
	body     ResolvingEffect
	ifPaid   bool
	required bool
}

func (e *sacrificePaymentEffect) Text() string {
	if e.required {
		return "mandatory sacrifice payment"
	}
	return "optional sacrifice payment"
}
func (e *sacrificePaymentEffect) Properties() EffectProperties {
	if nilDefinition(e.body) {
		return EffectProperties{}
	}
	return e.body.Properties()
}
func (e *sacrificePaymentEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "sacrifice-payment", Text: e.Text(), children: []definitionEdge{{"player", e.player}, {"subject", e.subject}, {"body", e.body}}}
}
func (e *sacrificePaymentEffect) Apply(ctx *EffectContext) error {
	subject, ok := e.subject.resolvePermanent(ctx)
	filter := NewPermanentFilter("declared permanent", func(*Permanent, *Game) bool { return false })
	if ok && !subject.PhasedOut {
		id, incarnation := subject.ID(), subject.incarnationID
		// The transaction must not select a returned object with the same card ID.
		filter = NewPermanentFilter("declared permanent", func(p *Permanent, _ *Game) bool { return p.ID() == id && p.incarnationID == incarnation })
	}
	payment := &unlessPaidEffect{player: e.player, costs: Costs(SacrificeMatchingIncludingSourceCost(filter, "Sacrifice the declared permanent")), effect: e.body, ifPaid: e.ifPaid, required: e.required}
	return payment.Apply(ctx)
}
