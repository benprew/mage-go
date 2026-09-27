package mage

import "github.com/benprew/mage-go/pkg/mage/core"

type playerDrawEventPattern struct {
	player       *PlayerBinding
	card         *CardBinding
	cardCapture  bool
	anyPlayer    bool
	opponentOnly bool
}

// PlayerDrawsCard saves the player who completes a card draw.
func PlayerDrawsCard(player *PlayerBinding) EventPattern {
	return playerDrawEventPattern{player: player, anyPlayer: true}
}

// ControllerDrawsCard matches a card drawn by the source controller.
// It saves the player at detection without declaring a target.
func ControllerDrawsCard(player *PlayerBinding) EventPattern {
	return playerDrawEventPattern{player: player}
}

// ControllerDrawsCardInto saves the drawn card for its current hand visit.
func ControllerDrawsCardInto(player *PlayerBinding, card *CardBinding) EventPattern {
	return playerDrawEventPattern{player: player, card: card, cardCapture: true}
}

// OpponentDrawsCard saves the opponent who drew a card when the draw occurs.
func OpponentDrawsCard(player *PlayerBinding) EventPattern {
	return playerDrawEventPattern{player: player, opponentOnly: true}
}

func (playerDrawEventPattern) eventPattern() {}
func (p playerDrawEventPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	if p.anyPlayer {
		return NewTriggered(core.EvtCardDrawn, false, effect)
	}
	condition := TriggerConditionData(EventPlayerIsController{})
	if p.opponentOnly {
		condition = EventPlayerIsOpponent{}
	}
	return NewTriggered(core.EvtCardDrawn, false, effect).SetConditionData(condition)
}
func (p playerDrawEventPattern) inspectDefinition() definitionNode {
	text := "you draw a card"
	if p.anyPlayer {
		text = "a player draws a card"
	} else if p.opponentOnly {
		text = "an opponent draws a card"
	}
	n := definitionNode{Role: "event-pattern", Text: text, battlefieldSource: true,
		bindingWrite: p.player, children: []definitionEdge{{"player", p.player}}}
	if p.card != nil {
		n.bindingWrites = []savedBinding{p.card}
		n.children = append(n.children, definitionEdge{"card", p.card})
	} else if p.cardCapture {
		n.problem = "supply a non-nil drawn card binding"
	}
	return n
}
func (p playerDrawEventPattern) captureEventEffects(event *core.GameEvent, effects []Effect) []Effect {
	captures := []ResolvingEffect{&eventPlayerCapture{p.player, event.PlayerID}}
	if p.cardCapture {
		captures = append(captures, &eventCardCapture{binding: p.card, value: capturedCard{id: event.DrawnCardID, visit: event.DrawnCardVisit, zone: core.ZoneHand}})
	}
	captures = append(captures, Sequence(effects...))
	return []Effect{Sequence(captures...)}
}
