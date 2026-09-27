package mage

type gainLifeEachEffect struct {
	subjects PlayerSet
	amount   ResolutionValue
}

// GainLifeEach gains life for each selected player in one instruction.
func GainLifeEach(subjects PlayerSet, amount ResolutionValue) ResolvingEffect {
	return &gainLifeEachEffect{subjects: subjects, amount: amount}
}

func (e *gainLifeEachEffect) Text() string { return "gain life for each player" }
func (*gainLifeEachEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit, Mass: true}
}
func (e *gainLifeEachEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subjects", e.subjects}, {"amount", e.amount}}}
}
func (e *gainLifeEachEffect) Apply(ctx *EffectContext) error {
	ids := e.subjects.resolvePlayers(ctx)
	amount := e.amount.resolveValue(ctx)
	players := make([]Player, 0, len(ids))
	for _, id := range ids {
		if player := ctx.Game.GetPlayer(id); player != nil {
			players = append(players, player)
		}
	}
	ctx.Game.playersGainLife(players, amount)
	return nil
}
