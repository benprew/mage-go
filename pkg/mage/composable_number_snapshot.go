package mage

type numberSnapshot struct {
	value   ResolutionValue
	binding *NumberBinding
}

// SnapshotNumber saves a value when this instruction resolves.
// Later instructions can replace the value in the same binding.
func SnapshotNumber(value ResolutionValue, binding *NumberBinding) ResolvingEffect {
	return &numberSnapshot{value: value, binding: binding}
}

func (*numberSnapshot) Text() string                 { return "save number" }
func (*numberSnapshot) Properties() EffectProperties { return EffectProperties{} }
func (e *numberSnapshot) Apply(ctx *EffectContext) error {
	amount := e.value.resolveValue(ctx)
	if ctx.Game.resolution.numberBindings == nil {
		ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
	}
	ctx.Game.resolution.numberBindings[e.binding] = amount
	return nil
}
func (e *numberSnapshot) inspectDefinition() definitionNode {
	return definitionNode{Role: "number-snapshot", bindingWrite: e.binding, children: []definitionEdge{{"value", e.value}, {"binding", e.binding}}}
}
