package mage

import "github.com/benprew/mage-go/pkg/mage/core"

type playerDiscardEventPattern struct {
	player       *PlayerBinding
	opponentOnly bool
}

// PlayerDiscardsCard saves the player who discards one card.
func PlayerDiscardsCard(player *PlayerBinding) EventPattern {
	return playerDiscardEventPattern{player: player}
}

// OpponentDiscardsCard saves the player who discards one card.
func OpponentDiscardsCard(player *PlayerBinding) EventPattern {
	return playerDiscardEventPattern{player: player, opponentOnly: true}
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
	return definitionNode{Role: "event-pattern", Text: label, battlefieldSource: true,
		bindingWrite: p.player, children: []definitionEdge{{"player", p.player}}}
}
func (p playerDiscardEventPattern) captureEventEffects(event *core.GameEvent, effects []Effect) []Effect {
	return []Effect{Sequence(&eventPlayerCapture{p.player, event.PlayerID}, Sequence(effects...))}
}
