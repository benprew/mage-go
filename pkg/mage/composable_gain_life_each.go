package mage

type gainLifeEachEffect struct {
	subjects PlayerSet
	amount   ResolutionValue
	lifeGainResultState
}

type lifeGainResultState struct {
	result  *NumberBinding
	count   int
	invalid bool
}

// LifeGainResultOption selects the life gained by one instruction.
type LifeGainResultOption interface {
	lifeGainResultOption()
	applyLifeGainResult(*lifeGainResultState)
}

type lifeGainedOption struct{ binding *NumberBinding }

// LifeGainedInto saves the life gained by the selected players.
func LifeGainedInto(binding *NumberBinding) LifeGainResultOption { return lifeGainedOption{binding} }
func (lifeGainedOption) lifeGainResultOption()                   {}
func (o lifeGainedOption) applyLifeGainResult(e *lifeGainResultState) {
	e.result = o.binding
	e.count++
	if o.binding == nil {
		e.invalid = true
	}
}

// GainLifeEach gains life for each selected player in one instruction.
func GainLifeEach(subjects PlayerSet, amount ResolutionValue, options ...LifeGainResultOption) ResolvingEffect {
	e := &gainLifeEachEffect{subjects: subjects, amount: amount}
	for _, option := range options {
		if nilDefinition(option) {
			e.invalid = true
			continue
		}
		option.applyLifeGainResult(&e.lifeGainResultState)
	}
	return e
}

func (e *gainLifeEachEffect) Text() string { return "gain life for each player" }
func (*gainLifeEachEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit, Mass: true}
}
func (e *gainLifeEachEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subjects", e.subjects}, {"amount", e.amount}}}
	if e.invalid {
		n.problem = "supply a non-nil life gain result binding"
	} else if e.count > 1 {
		n.problem = "supply the life gain result binding only once"
	}
	if e.result != nil {
		n.bindingWrite = e.result
		n.children = append(n.children, definitionEdge{"result", e.result})
	}
	return n
}
func (e *gainLifeEachEffect) Apply(ctx *EffectContext) error {
	ids := e.subjects.resolvePlayers(ctx)
	amount := e.amount.resolveValue(ctx)
	if e.result != nil {
		if ctx.Game.resolution.numberBindings == nil {
			ctx.Game.resolution.numberBindings = make(map[*NumberBinding]int)
		}
		ctx.Game.resolution.numberBindings[e.result] = 0
	}
	players := make([]Player, 0, len(ids))
	for _, id := range ids {
		if player := ctx.Game.GetPlayer(id); player != nil {
			players = append(players, player)
		}
	}
	gained := ctx.Game.playersGainLife(players, amount)
	if e.result != nil {
		ctx.Game.resolution.numberBindings[e.result] = gained
	}
	return nil
}
