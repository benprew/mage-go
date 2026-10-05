package mage

// AddPoisonFor adds poison counters to the selected player.
func AddPoisonFor(subject PlayerRef, amount ResolutionValue) ResolvingEffect {
	return &poisonPlayerEffect{subject: subject, amount: amount}
}

type poisonPlayerEffect struct {
	subject PlayerRef
	amount  ResolutionValue
}

func (*poisonPlayerEffect) Text() string { return "add poison counters to a player" }
func (*poisonPlayerEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *poisonPlayerEffect) Apply(ctx *EffectContext) error {
	id, ok := e.subject.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	amount := e.amount.resolveValue(ctx)
	if amount > 0 {
		if player := ctx.Game.GetPlayer(id); player != nil {
			player.AddPoisonCounters(amount)
		}
	}
	return nil
}
func (e *poisonPlayerEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "poison-player", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}, {"amount", e.amount}}}
}
