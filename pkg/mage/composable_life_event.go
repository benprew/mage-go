package mage

import "github.com/benprew/mage-go/pkg/mage/core"

type lifeLossEventPattern struct {
	player          *PlayerBinding
	amount          *NumberBinding
	captureAmount   bool
	duplicateAmount bool
	opponentOnly    bool
}

// OpponentLosesLife matches an opponent's life loss when it occurs.
// It saves the player without declaring a target.
func OpponentLosesLife(player *PlayerBinding) lifeLossEventPattern {
	return lifeLossEventPattern{player: player, opponentOnly: true}
}

// PlayerLosesLife saves any player who loses a positive amount of life.
func PlayerLosesLife(player *PlayerBinding) lifeLossEventPattern {
	return lifeLossEventPattern{player: player}
}

// WithAmount saves the life loss amount for the trigger body.
func (p lifeLossEventPattern) WithAmount(amount *NumberBinding) lifeLossEventPattern {
	if p.captureAmount {
		p.duplicateAmount = true
	}
	p.amount, p.captureAmount = amount, true
	return p
}

func (lifeLossEventPattern) eventPattern() {}
func (p lifeLossEventPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	if p.opponentOnly {
		return wheneverOpponentLosesLifeTrigger(effect, false).AndConditionData(EventAmountGreaterThan{N: 0})
	}
	return NewTriggered(core.EvtLifeLost, false, effect).SetConditionData(EventAmountGreaterThan{N: 0})
}
func (p lifeLossEventPattern) inspectDefinition() definitionNode {
	text := "a player loses life"
	if p.opponentOnly {
		text = "an opponent loses life"
	}
	n := definitionNode{Role: "event-pattern", Text: text, battlefieldSource: true,
		bindingWrite: p.player, children: []definitionEdge{{"player", p.player}}}
	if p.captureAmount {
		n.bindingWrites = append(n.bindingWrites, p.amount)
		n.children = append(n.children, definitionEdge{"amount", &eventAmountCapture{binding: p.amount}})
		if p.duplicateAmount {
			n.problem = "life loss event: supply one amount binding"
		}
	}
	return n
}
func (p lifeLossEventPattern) captureEventEffects(event *core.GameEvent, effects []Effect) []Effect {
	captures := []Effect{&eventPlayerCapture{p.player, event.PlayerID}}
	if p.captureAmount {
		captures = append(captures, &eventAmountCapture{p.amount, event.Amount})
	}
	captures = append(captures, Sequence(effects...))
	return []Effect{Sequence(captures...)}
}
