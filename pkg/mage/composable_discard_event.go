package mage

import "github.com/benprew/mage-go/pkg/mage/core"

type playerDiscardEventPattern struct {
	player       *PlayerBinding
	card         *CardBinding
	cardCaptures int
	opponentOnly bool
}

// PlayerDiscardsCard saves the player who discards one card.
func PlayerDiscardsCard(player *PlayerBinding) playerDiscardEventPattern {
	return playerDiscardEventPattern{player: player}
}

// OpponentDiscardsCard saves the player who discards one card.
func OpponentDiscardsCard(player *PlayerBinding) playerDiscardEventPattern {
	return playerDiscardEventPattern{player: player, opponentOnly: true}
}

// WithCard saves the discarded card only while its graveyard visit lasts.
func (p playerDiscardEventPattern) WithCard(card *CardBinding) playerDiscardEventPattern {
	p.card = card
	p.cardCaptures++
	return p
}

func (playerDiscardEventPattern) eventPattern() {}
func (p playerDiscardEventPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	trigger := NewTriggered(core.EvtDiscard, false, effect)
	if p.opponentOnly {
		trigger.SetConditionData(EventPlayerIsOpponent{})
	}
	return trigger
}
func (p playerDiscardEventPattern) inspectDefinition() definitionNode {
	label := "a player discards a card"
	if p.opponentOnly {
		label = "an opponent discards a card"
	}
	n := definitionNode{Role: "event-pattern", Text: label, battlefieldSource: true,
		bindingWrite: p.player, children: []definitionEdge{{"player", p.player}}}
	if p.cardCaptures > 1 {
		n.problem = "save the discarded card only once"
	} else if p.cardCaptures == 1 && p.card == nil {
		n.problem = "supply a non-nil discarded card binding"
	} else if p.card != nil {
		n.bindingWrites = []savedBinding{p.card}
		n.children = append(n.children, definitionEdge{"card", p.card})
	}
	return n
}
func (p playerDiscardEventPattern) captureEventEffects(event *core.GameEvent, effects []Effect) []Effect {
	captures := []ResolvingEffect{&eventPlayerCapture{p.player, event.PlayerID}}
	if p.cardCaptures == 1 {
		captures = append(captures, &eventCardCapture{binding: p.card, value: capturedCard{id: event.SourceID, visit: event.DiscardedCardVisit, zone: core.ZoneGraveyard}})
	}
	captures = append(captures, Sequence(effects...))
	return []Effect{Sequence(captures...)}
}
