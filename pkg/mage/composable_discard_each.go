package mage

type discardEachEffect struct{ subjects PlayerSet }

// DiscardEach makes the selected players discard their hands together.
func DiscardEach(subjects PlayerSet) ResolvingEffect {
	return &discardEachEffect{subjects: subjects}
}

func (e *discardEachEffect) Text() string { return "each selected player discards their hand" }
func (*discardEachEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment, Mass: true}
}
func (e *discardEachEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subjects", e.subjects}}}
}
func (e *discardEachEffect) Apply(ctx *EffectContext) error {
	ctx.Game.discardHandsTogether(e.subjects.resolvePlayers(ctx), ctx.SourceID)
	return nil
}
