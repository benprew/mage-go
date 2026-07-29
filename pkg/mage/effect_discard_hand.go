package mage

// DiscardHandEffect discards the complete hand of selected players.
type DiscardHandEffect struct {
	selector PlayerSelector
}

// DiscardHand discards every card in the selected players' hands as an
// effect. It defaults to the effect controller.
func DiscardHand() *DiscardHandEffect {
	return &DiscardHandEffect{}
}

func (e *DiscardHandEffect) Targeting(selector PlayerSelector) *DiscardHandEffect {
	e.selector = selector
	return e
}

func (*DiscardHandEffect) Text() string { return "discard your hand" }
func (*DiscardHandEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}

func (e *DiscardHandEffect) Apply(ctx *EffectContext) error {
	selector := e.selector
	if selector == nil {
		selector = SelectController()
	}
	for _, playerID := range selector.Select(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets) {
		player := ctx.Game.GetPlayer(playerID)
		if player == nil {
			continue
		}
		for _, card := range append([]Card(nil), player.Hand()...) {
			ctx.Game.PlayerDiscardByEffect(player, card.ID(), ctx.SourceID)
		}
	}
	return nil
}
