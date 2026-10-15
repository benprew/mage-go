package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// discardHandsTogether moves the selected hands before it sends discard events.
func (g *Game) discardHandsTogether(ids []uuid.UUID, sourceID uuid.UUID) int {
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
	return g.discardActionsTogether(actions)
}

// discardActionsTogether keeps each event after all card moves.
func (g *Game) discardActionsTogether(actions []*DiscardAction) int {
	return len(g.discardActionsTogetherWithEvents(actions, false))
}

func (g *Game) discardActionsTogetherWithEvents(actions []*DiscardAction, requireAll bool) []core.GameEvent {
	var prepared []*DiscardAction
	for _, action := range actions {
		result := g.effects.ApplyReplacements(action, g)
		if discard, ok := result.(*DiscardAction); ok {
			// A hidden destination removes the names required by this cost.
			if action.requiresCharacteristics && discard.Destination() == core.ZoneLibrary {
				return nil
			}
			if action.handVisit != uuid.Nil {
				proposal := *discard
				proposal.handVisit = action.handVisit
				discard = &proposal
			}
			prepared = append(prepared, discard)
		}
	}
	if requireAll && len(prepared) != len(actions) {
		return nil
	}
	g.orderDiscardDestinations(prepared, core.ZoneGraveyard)
	g.orderDiscardDestinations(prepared, core.ZoneLibrary)
	var events []core.GameEvent
	for _, action := range prepared {
		if card, ok := g.moveDiscardedCard(action); ok {
			events = append(events, core.GameEvent{Type: core.EvtDiscard, SourceID: card.ID(), PlayerID: action.PlayerID(), DiscardedCardVisit: g.graveyardVisit(card.ID())})
		}
	}
	for _, event := range events {
		g.FireEvent(event)
	}
	return events
}

func (g *Game) orderDiscardDestinations(actions []*DiscardAction, destination core.Zone) {
	for offset := range len(g.players) {
		player := g.players[(g.turns.ActivePlayerIndex()+offset)%len(g.players)]
		var cards []Card
		var positions []int
		byID := make(map[uuid.UUID]*DiscardAction)
		for index, action := range actions {
			if action.PlayerID() != player.PlayerID() || action.Destination() != destination {
				continue
			}
			for _, card := range player.Hand() {
				if card.ID() == action.CardID() {
					cards = append(cards, card)
					positions = append(positions, index)
					byID[card.ID()] = action
					break
				}
			}
		}
		if len(cards) < 2 {
			continue
		}
		var ordered []Card
		if destination == core.ZoneLibrary {
			ordered = g.orderCardsByChoice(player, cards, "choose the next card for the top of your library")
		} else {
			ordered = g.orderCardsForGraveyard(player, cards)
		}
		for index, card := range ordered {
			position := index
			if destination == core.ZoneLibrary {
				position = len(ordered) - 1 - index
			}
			actions[positions[position]] = byID[card.ID()]
		}
	}
}
