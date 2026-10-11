package mage

// MayDrawCard lets the selected player decide whether to draw at resolution.
func MayDrawCard(player PlayerRef) ResolvingEffect { return &optionalDrawCard{player: player} }

type optionalDrawCard struct{ player PlayerRef }

func (*optionalDrawCard) Text() string                 { return "may draw a card" }
func (*optionalDrawCard) Properties() EffectProperties { return drawAmountProperties(Fixed(1)) }
func (e *optionalDrawCard) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"player", e.player}}}
}
func (e *optionalDrawCard) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player)
}
func (e *optionalDrawCard) Apply(ctx *EffectContext) error {
	id, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	player := ctx.Game.GetPlayer(id)
	if player != nil && player.ChooseMayAbility(e.Text()) {
		ctx.Game.PlayerDrawCard(player)
	}
	return nil
}
