package mage

import "github.com/benprew/mage-go/pkg/mage/core"

type opponentDiscardEventPattern struct{ player *PlayerBinding }

// OpponentDiscardsCard saves the player who discards one card.
func OpponentDiscardsCard(player *PlayerBinding) EventPattern {
	return opponentDiscardEventPattern{player: player}
}

func (opponentDiscardEventPattern) eventPattern() {}
func (p opponentDiscardEventPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(core.EvtDiscard, false, effect).SetConditionData(EventPlayerIsOpponent{})
}
func (p opponentDiscardEventPattern) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-pattern", Text: "an opponent discards a card", battlefieldSource: true,
		bindingWrite: p.player, children: []definitionEdge{{"player", p.player}}}
}
func (p opponentDiscardEventPattern) captureEventEffects(event *core.GameEvent, effects []Effect) []Effect {
	return []Effect{Sequence(&eventPlayerCapture{p.player, event.PlayerID}, Sequence(effects...))}
}
