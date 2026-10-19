package mage

// MayCastHandCardFaceDownWithoutPaying offers a separate choice to cast a saved
// hand visit. Declining the cast does not change the card or spend mana.
func MayCastHandCardFaceDownWithoutPaying(player PlayerRef, card CardRef) ResolvingEffect {
	return &optionalFaceDownCast{faceDownCastEffect{player: player, card: card}}
}

type optionalFaceDownCast struct{ faceDownCastEffect }

func (*optionalFaceDownCast) Text() string { return "may cast the chosen card face down" }
func (e *optionalFaceDownCast) Apply(ctx *EffectContext) error {
	e.clearResult(ctx)
	id, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	player := ctx.Game.GetPlayer(id)
	card, ok := e.card.resolveCard(ctx)
	if player == nil || !ok {
		return nil
	}
	if !ctx.Game.canCastHandCardFaceDownWithoutPaying(id, card.ID()) {
		return nil
	}
	if !player.ChooseMayAbility(e.Text()) {
		return nil
	}
	// The decision can change the game. Check the saved visit and costs again.
	return e.faceDownCastEffect.Apply(ctx)
}
