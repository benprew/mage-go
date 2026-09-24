package mage

// NumberBinding declares a saved number. Each resolution has its own value.
type NumberBinding struct{ name string }

// BindNumber declares a number binding with a diagnostic name.
func BindNumber(name string) *NumberBinding { return &NumberBinding{name: name} }

// FromNumber reads a saved number. An absent binding supplies zero.
func FromNumber(binding *NumberBinding) ResolutionValue { return savedNumberValue{binding} }

type savedNumberValue struct{ binding *NumberBinding }

func (v savedNumberValue) Text() string {
	if v.binding == nil {
		return "saved number"
	}
	return v.binding.name
}
func (v savedNumberValue) resolveValue(ctx *EffectContext) int {
	return ctx.Game.resolution.numberBindings[v.binding]
}
func (v savedNumberValue) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-value", Text: v.Text(), bindingRead: v.binding, children: []definitionEdge{{"binding", v.binding}}}
}
func (b *NumberBinding) bindingName() string { return b.name }
func (b *NumberBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "number-binding", Text: b.name, bindingDeclaration: b}
	if b.name == "" {
		n.problem = "supply a binding name"
	}
	return n
}

type eventAmountCapture struct {
	binding *NumberBinding
	amount  int
}

func (*eventAmountCapture) Text() string                 { return "remember the event amount" }
func (*eventAmountCapture) Properties() EffectProperties { return EffectProperties{} }
func (e *eventAmountCapture) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.numberBindings == nil {
		ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
	}
	ctx.Game.resolution.numberBindings[e.binding] = e.amount
	return nil
}
func (e *eventAmountCapture) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-amount-capture", bindingWrite: e.binding, children: []definitionEdge{{"amount", e.binding}}}
}
