package mage

type permanentControlExchangeEffect struct {
	first, second PermanentRef
}

// ExchangePermanentControl exchanges current controllers of two present visits.
// If either visit cannot change control, both controllers stay the same.
func ExchangePermanentControl(first, second PermanentRef) ResolvingEffect {
	return &permanentControlExchangeEffect{first: first, second: second}
}

func (*permanentControlExchangeEffect) Text() string                 { return "exchange control of two permanents" }
func (*permanentControlExchangeEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *permanentControlExchangeEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"first", e.first}, {"second", e.second}}}
}
func (e *permanentControlExchangeEffect) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.first) || permanentRefReadsSourcePermanent(e.second)
}
func (e *permanentControlExchangeEffect) Apply(ctx *EffectContext) error {
	first, firstOK := e.first.resolvePermanent(ctx)
	second, secondOK := e.second.resolvePermanent(ctx)
	if firstOK && secondOK {
		ctx.Game.ExchangeControl(first.ID(), second.ID(), ctx.SourceID)
	}
	return nil
}
