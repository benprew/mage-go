package mage

type cardOwnershipExchangeEffect struct {
	first, second CardRef
}

// ExchangeCardOwnership exchanges current owners without moving either card.
// Both cards are checked before the changes. ChangeOwner sends no events.
func ExchangeCardOwnership(first, second CardRef) ResolvingEffect {
	return &cardOwnershipExchangeEffect{first: first, second: second}
}

func (*cardOwnershipExchangeEffect) Text() string                 { return "exchange ownership of two cards" }
func (*cardOwnershipExchangeEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *cardOwnershipExchangeEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", children: []definitionEdge{{"first", e.first}, {"second", e.second}}}
}
func (e *cardOwnershipExchangeEffect) Apply(ctx *EffectContext) error {
	g := ctx.Game
	if !g.anteEnabled || g.anteSettled {
		return nil
	}
	first, ok := e.first.resolveCard(ctx)
	if !ok || first.IsToken() {
		return nil
	}
	second, ok := e.second.resolveCard(ctx)
	if !ok || second.IsToken() {
		return nil
	}
	firstOwner, secondOwner := first.Owner(), second.Owner()
	if first.ID() == second.ID() || firstOwner == secondOwner || g.GetPlayer(firstOwner) == nil || g.GetPlayer(secondOwner) == nil {
		return nil
	}
	if err := g.ChangeOwner(first.ID(), secondOwner); err != nil {
		return err
	}
	return g.ChangeOwner(second.ID(), firstOwner)
}
