package mage

type discardCardEffect struct{ card CardRef }

// DiscardCard discards a referenced card if it remains in its owner's hand.
func DiscardCard(card CardRef) ResolvingEffect { return &discardCardEffect{card: card} }

func (*discardCardEffect) Text() string { return "discard the referenced card" }
func (*discardCardEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *discardCardEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"card", e.card}}}
}
func (e *discardCardEffect) Apply(ctx *EffectContext) error {
	card, ok := e.card.resolveCard(ctx)
	if !ok {
		return nil
	}
	owner := ctx.Game.GetPlayer(card.Owner())
	if owner == nil {
		return nil
	}
	for _, held := range owner.Hand() {
		if held.ID() == card.ID() {
			ctx.Game.PlayerDiscardByEffect(owner, card.ID(), ctx.SourceID)
			return nil
		}
	}
	return nil
}
