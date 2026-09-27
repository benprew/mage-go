package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// discardHandsTogether moves the selected hands before it sends discard events.
func (g *Game) discardHandsTogether(ids []uuid.UUID, sourceID uuid.UUID) {
	var actions []*DiscardAction
	seen := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		player := g.GetPlayer(id)
		if player == nil {
			continue
		}
		for _, card := range append([]Card(nil), player.Hand()...) {
			actions = append(actions, NewDiscardAction(sourceID, id, card.ID()))
		}
	}
	var prepared []*DiscardAction
	for _, action := range actions {
		result := g.effects.ApplyReplacements(action, g)
		if discard, ok := result.(*DiscardAction); ok {
			prepared = append(prepared, discard)
		}
	}
	var events []core.GameEvent
	for _, action := range prepared {
		if card, ok := g.moveDiscardedCard(action); ok {
			events = append(events, core.GameEvent{Type: core.EvtDiscard, SourceID: card.ID(), PlayerID: action.PlayerID()})
		}
	}
	for _, event := range events {
		g.FireEvent(event)
	}
}
