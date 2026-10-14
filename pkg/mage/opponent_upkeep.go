package mage

import "github.com/benprew/mage-go/pkg/mage/core"

type opponentUpkeepPattern struct{ player *PlayerBinding }

// OpponentUpkeep saves the upkeep player at detection.
// Later control changes cannot change the saved player.
func OpponentUpkeep(player *PlayerBinding) EventPattern {
	return opponentUpkeepPattern{player: player}
}

func (opponentUpkeepPattern) eventPattern() {}

func (opponentUpkeepPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return EachUpkeep.makeStepTrigger(effect).AndConditionData(EventPlayerIsOpponent{})
}

func (p opponentUpkeepPattern) inspectDefinition() definitionNode {
	n := (turnStepEventPattern{EachUpkeep, p.player}).inspectDefinition()
	n.Text = "beginning of opponent's upkeep"
	return n
}

func (p opponentUpkeepPattern) captureEventEffects(event *core.GameEvent, effects []Effect) []Effect {
	return (turnStepEventPattern{EachUpkeep, p.player}).captureEventEffects(event, effects)
}
