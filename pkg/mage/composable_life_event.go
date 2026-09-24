package mage

import "github.com/benprew/mage-go/pkg/mage/core"

type opponentLifeEventPattern struct {
	player        *PlayerBinding
	amount        *NumberBinding
	captureAmount bool
}

// OpponentLosesLife matches an opponent's life loss when it occurs.
// It saves the player without declaring a target.
func OpponentLosesLife(player *PlayerBinding) opponentLifeEventPattern {
	return opponentLifeEventPattern{player: player}
}

// WithAmount saves the life loss amount for the trigger body.
func (p opponentLifeEventPattern) WithAmount(amount *NumberBinding) EventPattern {
	p.amount, p.captureAmount = amount, true
	return p
}

func (opponentLifeEventPattern) eventPattern() {}
func (opponentLifeEventPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return wheneverOpponentLosesLifeTrigger(effect, false).AndConditionData(EventAmountGreaterThan{N: 0})
}
func (p opponentLifeEventPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "an opponent loses life", battlefieldSource: true,
		bindingWrite: p.player, children: []definitionEdge{{"player", p.player}}}
	if p.captureAmount {
		n.children = append(n.children, definitionEdge{"amount", &eventAmountCapture{binding: p.amount}})
	}
	return n
}
func (p opponentLifeEventPattern) captureEventEffects(event *core.GameEvent, effects []Effect) []Effect {
	captures := []Effect{&eventPlayerCapture{p.player, event.PlayerID}}
	if p.captureAmount {
		captures = append(captures, &eventAmountCapture{p.amount, event.Amount})
	}
	captures = append(captures, Sequence(effects...))
	return []Effect{Sequence(captures...)}
}
