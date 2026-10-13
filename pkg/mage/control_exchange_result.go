package mage

// ControlExchangeResultBinding keeps one exchange outcome in a resolution frame.
type ControlExchangeResultBinding struct{ value *NumberBinding }

// BindControlExchangeResult declares a separate saved exchange outcome.
func BindControlExchangeResult(name string) *ControlExchangeResultBinding {
	return &ControlExchangeResultBinding{value: BindNumber(name)}
}

func controlExchangeResultNumber(result *ControlExchangeResultBinding) *NumberBinding {
	if result == nil {
		return nil
	}
	return result.value
}

// ExchangePermanentControlInto saves success or failure after an atomic exchange.
func ExchangePermanentControlInto(first, second PermanentRef, result *ControlExchangeResultBinding) ResolvingEffect {
	return &permanentControlExchangeEffect{first: first, second: second, result: controlExchangeResultNumber(result), captureResult: true}
}

// IfControlExchanged runs its body only after the saved exchange succeeds.
func IfControlExchanged(result *ControlExchangeResultBinding, body ResolvingEffect) ResolvingEffect {
	return &controlExchangeResultBranch{result: controlExchangeResultNumber(result), body: body}
}

type controlExchangeResultBranch struct {
	result *NumberBinding
	body   ResolvingEffect
}

func (*controlExchangeResultBranch) Text() string { return "if control was exchanged" }
func (e *controlExchangeResultBranch) Properties() EffectProperties {
	if nilDefinition(e.body) {
		return EffectProperties{}
	}
	return e.body.Properties()
}
func (e *controlExchangeResultBranch) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.numberBindings[e.result] != 0 {
		return e.body.Apply(ctx)
	}
	return nil
}
func (e *controlExchangeResultBranch) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-branch", Text: e.Text(), children: []definitionEdge{{"result", FromNumber(e.result)}, {"then", e.body}}}
}
func (e *controlExchangeResultBranch) refersToSourcePermanent() bool {
	return effectReadsSourcePermanent(e.body)
}
