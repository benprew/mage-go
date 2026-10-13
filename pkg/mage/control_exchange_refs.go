package mage

type permanentControlExchangeEffect struct {
	first, second PermanentRef
	result        *NumberBinding
	captureResult bool
}

// ExchangePermanentControl exchanges current controllers of two present visits.
// If either visit cannot change control, both controllers stay the same.
func ExchangePermanentControl(first, second PermanentRef) ResolvingEffect {
	return &permanentControlExchangeEffect{first: first, second: second}
}

func (*permanentControlExchangeEffect) Text() string                 { return "exchange control of two permanents" }
func (*permanentControlExchangeEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *permanentControlExchangeEffect) inspectDefinition() definitionNode {
	node := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"first", e.first}, {"second", e.second}}}
	if e.captureResult {
		node.bindingWrite = e.result
		node.children = append(node.children, definitionEdge{"result", e.result})
	}
	return node
}
func (e *permanentControlExchangeEffect) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.first) || permanentRefReadsSourcePermanent(e.second)
}
func (e *permanentControlExchangeEffect) Apply(ctx *EffectContext) error {
	first, firstOK := e.first.resolvePermanent(ctx)
	second, secondOK := e.second.resolvePermanent(ctx)
	success := 0
	if firstOK && secondOK && ctx.Game.ExchangeControl(first.ID(), second.ID(), ctx.SourceID) {
		success = 1
	}
	if e.captureResult {
		return SnapshotNumber(Fixed(success), e.result).Apply(ctx)
	}
	return nil
}
