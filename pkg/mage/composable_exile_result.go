package mage

// ExileResultOption saves the outcome of one battlefield exile instruction.
type ExileResultOption interface {
	exileResultOption()
	applyExileResult(*exileResultEffect)
}

type exiledThisWayOption struct{ binding *NumberBinding }

// ExiledThisWayInto saves 1 when the subject moves from the battlefield.
func ExiledThisWayInto(binding *NumberBinding) ExileResultOption {
	return exiledThisWayOption{binding: binding}
}
func (exiledThisWayOption) exileResultOption() {}
func (o exiledThisWayOption) applyExileResult(e *exileResultEffect) {
	e.result = o.binding
	e.optionCount++
	if o.binding == nil {
		e.invalidOption = true
	}
}

type exileResultEffect struct {
	subject       PermanentRef
	result        *NumberBinding
	optionCount   int
	invalidOption bool
}

// ExileWithResult exiles one permanent and saves the completed move.
func ExileWithResult(subject PermanentRef, options ...ExileResultOption) ResolvingEffect {
	e := &exileResultEffect{subject: subject}
	for _, option := range options {
		if nilDefinition(option) {
			e.invalidOption = true
			continue
		}
		option.applyExileResult(e)
	}
	return e
}

func (*exileResultEffect) Text() string { return "exile permanent with result" }
func (*exileResultEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *exileResultEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
	if e.invalidOption || e.result == nil {
		n.problem = "supply a non-nil exile result binding"
	} else if e.optionCount != 1 {
		n.problem = "supply the exile result binding only once"
	}
	if e.result != nil {
		n.bindingWrite = e.result
		n.children = append(n.children, definitionEdge{"result", e.result})
	}
	return n
}
func (e *exileResultEffect) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.numberBindings == nil {
		ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
	}
	ctx.Game.resolution.numberBindings[e.result] = 0
	if p, ok := e.subject.resolvePermanent(ctx); ok {
		ctx.Game.ExilePermanent(p)
		ctx.Game.resolution.numberBindings[e.result] = 1
	}
	return nil
}
