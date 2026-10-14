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
	var owned []capturedCard
	for _, subject := range selected {
		for _, card := range player.Graveyard() {
			if card.ID() == subject.id {
				owned = append(owned, subject)
				break
			}
		}
	}
	g.exileSelectedGraveyardVisitsTogether(owned, sourceID)
}
