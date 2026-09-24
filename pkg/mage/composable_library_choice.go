package mage

type libraryChoiceEffect struct {
	amount  ResolutionValue
	surveil bool
}

// ScryAtResolution reads the amount in the current frame, then scries for You.
func ScryAtResolution(amount ResolutionValue) ResolvingEffect {
	return &libraryChoiceEffect{amount: amount}
}

// SurveilAtResolution reads the amount in the current frame, then surveils for You.
func SurveilAtResolution(amount ResolutionValue) ResolvingEffect {
	return &libraryChoiceEffect{amount: amount, surveil: true}
}

func (e *libraryChoiceEffect) verb() string {
	if e.surveil {
		return "surveil"
	}
	return "scry"
}
func (e *libraryChoiceEffect) Text() string {
	if amount, ok := e.amount.(fixedValue); ok {
		return e.verb() + " " + amount.Text()
	}
	return e.verb()
}
func (*libraryChoiceEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *libraryChoiceEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: e.verb(), Text: e.Text(), children: []definitionEdge{{"amount", e.amount}}}
}
func (e *libraryChoiceEffect) Apply(ctx *EffectContext) error {
	amount := Fixed(e.amount.resolveValue(ctx))
	if e.surveil {
		return Surveil(amount).Apply(ctx)
	}
	return Scry(amount).Apply(ctx)
}
