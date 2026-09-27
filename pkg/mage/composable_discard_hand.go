package mage

type discardEntireHandEffect struct {
	subject PlayerRef
	result  *NumberBinding
	count   int
	invalid bool
}

// DiscardEntireHand makes one player discard the hand they have at resolution.
func DiscardEntireHand(subject PlayerRef, options ...DiscardResultOption) ResolvingEffect {
	e := &discardEntireHandEffect{subject: subject}
	for _, option := range options {
		if nilDefinition(option) {
			e.invalid = true
			continue
		}
		option.applyDiscardResult(e)
	}
	return e
}

// DiscardResultOption selects a result from one hand-discard instruction.
type DiscardResultOption interface {
	discardResultOption()
	applyDiscardResult(*discardEntireHandEffect)
}

type discardAmountOption struct{ binding *NumberBinding }

// DiscardAmountInto saves the number of cards discarded by this instruction.
func DiscardAmountInto(binding *NumberBinding) DiscardResultOption {
	return discardAmountOption{binding: binding}
}
func (discardAmountOption) discardResultOption() {}
func (o discardAmountOption) applyDiscardResult(e *discardEntireHandEffect) {
	e.result = o.binding
	e.count++
	if o.binding == nil {
		e.invalid = true
	}
}

func (e *discardEntireHandEffect) Text() string {
	return "discard entire hand for " + InspectDefinition(e.subject).Text
}
func (*discardEntireHandEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *discardEntireHandEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
	if e.invalid {
		n.problem = "supply a non-nil discard result binding"
	} else if e.count > 1 {
		n.problem = "supply the discard result binding only once"
	}
	if e.result != nil {
		n.bindingWrite = e.result
		n.children = append(n.children, definitionEdge{"result", e.result})
	}
	return n
}
func (e *discardEntireHandEffect) Apply(ctx *EffectContext) error {
	if e.result != nil {
		if ctx.Game.resolution.numberBindings == nil {
			ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
		}
		ctx.Game.resolution.numberBindings[e.result] = 0
	}
	id, ok := e.subject.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	player := ctx.Game.GetPlayer(id)
	if player == nil {
		return nil
	}
	count := discardPlayerHand(ctx.Game, player, ctx.SourceID)
	if e.result != nil {
		ctx.Game.resolution.numberBindings[e.result] = count
	}
	return nil
}
