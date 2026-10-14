package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// ExileGraveyardFor reads one player's graveyard when the effect resolves.
// All saved visits move before any zone event is sent.
func ExileGraveyardFor(player PlayerRef) ResolvingEffect {
	return &graveyardBatchExile{player: player}
}

type graveyardBatchExile struct{ player PlayerRef }

func (*graveyardBatchExile) Text() string { return "exile all cards from a player's graveyard" }
func (*graveyardBatchExile) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *graveyardBatchExile) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"player", e.player}}}
}
func (e *graveyardBatchExile) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player)
}
func (e *graveyardBatchExile) Apply(ctx *EffectContext) error {
	id, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	player := ctx.Game.GetPlayer(id)
	if player == nil {
		return nil
	}
	selected := make([]capturedCard, 0, len(player.Graveyard()))
	for _, card := range player.Graveyard() {
		selected = append(selected, capturedCard{id: card.ID(), visit: ctx.Game.graveyardVisit(card.ID()), zone: core.ZoneGraveyard})
	}
	ctx.Game.exileGraveyardVisitsTogether(id, selected, ctx.SourceID)
	return nil
}

func (g *Game) exileGraveyardVisitsTogether(playerID uuid.UUID, selected []capturedCard, sourceID uuid.UUID) {
	player := g.GetPlayer(playerID)
	if player == nil {
		return
	}
	removed := make([]Card, 0, len(selected))
	for _, subject := range selected {
		if subject.zone != core.ZoneGraveyard || subject.visit == uuid.Nil || g.graveyardVisit(subject.id) != subject.visit {
			continue
		}
		if card, ok := player.RemoveFromGraveyard(subject.id); ok {
			removed = append(removed, card)
		}
	}
	for _, card := range removed {
		g.ExileCard(card, sourceID)
	}
	for _, card := range removed {
		g.FireEvent(core.GameEvent{Type: core.EvtZoneChange, SourceID: card.ID(), PlayerID: playerID, FromZone: core.ZoneGraveyard, ToZone: core.ZoneExile})
	}
	if len(removed) > 0 {
		g.FireEvent(core.GameEvent{Type: core.EvtCardsLeftGraveyard, PlayerID: playerID, Amount: len(removed)})
	}
}
