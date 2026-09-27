package mage

import "github.com/google/uuid"

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
		discardPlayerHand(ctx.Game, player, ctx.SourceID)
	}
	return nil
}

func discardPlayerHand(game *Game, player Player, sourceID uuid.UUID) int {
	count := 0
	for _, card := range append([]Card(nil), player.Hand()...) {
		if _, discarded := game.PlayerDiscardByEffect(player, card.ID(), sourceID); discarded {
			count++
		}
	}
	return count
}
