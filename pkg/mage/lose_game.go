package mage

// LoseGameFor records game loss for the selected player.
// A life-total loss exemption cannot prevent this loss.
func LoseGameFor(subject PlayerRef) ResolvingEffect {
	return &loseGameEffect{subject: subject}
}

type loseGameEffect struct{ subject PlayerRef }

func (*loseGameEffect) Text() string { return "lose the game" }
func (*loseGameEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *loseGameEffect) Apply(ctx *EffectContext) error {
	if player := handOperationPlayer(e.subject, ctx); player != nil {
		player.SetLost()
	}
	return nil
}
func (e *loseGameEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.subject)
}
func (e *loseGameEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "lose-game", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
}
