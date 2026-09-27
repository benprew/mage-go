package mage

type discardEachEffect struct {
	subjects    PlayerSet
	result      *NumberBinding
	resultCount int
	invalid     bool
}

type DiscardEachResultOption interface {
	discardEachResultOption()
	applyDiscardEachResult(*discardEachEffect)
}

type discardTotalOption struct{ binding *NumberBinding }

// DiscardTotalInto saves the number of completed discards in one batch.
func DiscardTotalInto(binding *NumberBinding) DiscardEachResultOption {
	return discardTotalOption{binding}
}
func (discardTotalOption) discardEachResultOption() {}
func (o discardTotalOption) applyDiscardEachResult(e *discardEachEffect) {
	e.result = o.binding
	e.resultCount++
	if o.binding == nil {
		e.invalid = true
	}
}

// DiscardEach makes the selected players discard their hands together.
func DiscardEach(subjects PlayerSet, options ...DiscardEachResultOption) ResolvingEffect {
	e := &discardEachEffect{subjects: subjects}
	for _, option := range options {
		if nilDefinition(option) {
			e.invalid = true
			continue
		}
		option.applyDiscardEachResult(e)
	}
	return e
}

func (e *discardEachEffect) Text() string { return "each selected player discards their hand" }
func (*discardEachEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Mass: true}
}
func (e *discardEachEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subjects", e.subjects}}}
	if e.invalid {
		n.problem = "supply a non-nil discard total binding"
	} else if e.resultCount > 1 {
		n.problem = "supply the discard total binding only once"
	}
	if e.result != nil {
		n.bindingWrite = e.result
		n.children = append(n.children, definitionEdge{"result", e.result})
	}
	return n
}
func (e *discardEachEffect) Apply(ctx *EffectContext) error {
	if e.result != nil {
		if ctx.Game.resolution.numberBindings == nil {
			ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
		}
		ctx.Game.resolution.numberBindings[e.result] = 0
	}
	count := ctx.Game.discardHandsTogether(e.subjects.resolvePlayers(ctx), ctx.SourceID)
	if e.result != nil {
		ctx.Game.resolution.numberBindings[e.result] = count
	}
	return nil
}
