package mage

// IfPositive runs the body when the resolved value is greater than zero.
func IfPositive(value ResolutionValue, body ResolvingEffect) ResolvingEffect {
	return &positiveBranch{value: value, body: body}
}

type positiveBranch struct {
	value ResolutionValue
	body  ResolvingEffect
}

func (e *positiveBranch) Text() string                 { return "if positive, " + e.body.Text() }
func (e *positiveBranch) Properties() EffectProperties { return e.body.Properties() }
func (e *positiveBranch) Apply(ctx *EffectContext) error {
	if e.value.resolveValue(ctx) > 0 {
		return e.body.Apply(ctx)
	}
	return nil
}
func (e *positiveBranch) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolution-branch", Text: "if positive", children: []definitionEdge{{"value", e.value}, {"then", e.body}}}
}
