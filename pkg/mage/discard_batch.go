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
	events, _ := g.discardActionsTogetherWithFacts(actions, requireAll, nil)
	return events
}

func (g *Game) discardActionsTogetherWithFacts(actions []*DiscardAction, requireAll bool, facts []CardPaymentFact) (events []core.GameEvent, values []int) {
	var prepared []*DiscardAction
	for _, action := range actions {
		g.snapshotDiscardCause(action)
	}
	for _, action := range actions {
		result := g.effects.ApplyReplacements(action, g)
		if discard, ok := result.(*DiscardAction); ok {
			// A hidden destination removes the names required by this cost.
			if action.requiresCharacteristics && discard.Destination() == core.ZoneLibrary {
				return nil, nil
			}
			if action.handVisit != uuid.Nil {
				proposal := *discard
				proposal.handVisit = action.handVisit
				discard = &proposal
			}
			discard.discardEvent = action.discardEvent
			prepared = append(prepared, discard)
		}
	}
	if requireAll && len(prepared) != len(actions) {
		return nil, nil
	}
	g.orderDiscardDestinations(prepared, core.ZoneGraveyard)
	g.orderDiscardDestinations(prepared, core.ZoneLibrary)
	values = make([]int, len(facts))
	for _, action := range prepared {
		matches := make([]int, len(facts))
		if len(facts) > 0 && (action.Destination() == core.ZoneGraveyard || action.Destination() == core.ZoneExile) {
			for _, card := range g.GetPlayer(action.PlayerID()).Hand() {
				if card.ID() == action.CardID() {
					for i, fact := range facts {
						if fact.Filter.Match(card) {
							matches[i] = 1
						}
					}
					break
				}
			}
		}
		if _, ok := g.moveDiscardedCard(action); ok {
			for i, value := range matches {
				values[i] += value
			}
			events = append(events, g.completedDiscardEvent(action))
		}
	}
	for _, event := range events {
		g.FireEvent(event)
	}
	return events, values
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
