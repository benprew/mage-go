package mage

import "github.com/benprew/mage-go/pkg/mage/core"

type playerDrawEventPattern struct {
	player       *PlayerBinding
	opponentOnly bool
}

// ControllerDrawsCard matches a card drawn by the source controller.
// It saves the player at detection without declaring a target.
func ControllerDrawsCard(player *PlayerBinding) EventPattern {
	return playerDrawEventPattern{player: player}
}

// OpponentDrawsCard saves the opponent who drew a card when the draw occurs.
func OpponentDrawsCard(player *PlayerBinding) EventPattern {
	return playerDrawEventPattern{player: player, opponentOnly: true}
}

func (playerDrawEventPattern) eventPattern() {}
func (p playerDrawEventPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	condition := TriggerConditionData(EventPlayerIsController{})
	if p.opponentOnly {
		condition = EventPlayerIsOpponent{}
	}
	return NewTriggered(core.EvtCardDrawn, false, effect).SetConditionData(condition)
}
func (p playerDrawEventPattern) inspectDefinition() definitionNode {
	text := "you draw a card"
	if p.opponentOnly {
		text = "an opponent draws a card"
	}
	return definitionNode{Role: "event-pattern", Text: text, battlefieldSource: true,
		bindingWrite: p.player, children: []definitionEdge{{"player", p.player}}}
}
func (p playerDrawEventPattern) captureEventEffects(event *core.GameEvent, effects []Effect) []Effect {
	return []Effect{Sequence(&eventPlayerCapture{p.player, event.PlayerID}, Sequence(effects...))}
}
