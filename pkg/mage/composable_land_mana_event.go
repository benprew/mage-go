package mage

import . "github.com/benprew/mage-go/pkg/mage/core"

type landTappedForManaPattern struct {
	player          *PlayerBinding
	amount          *NumberBinding
	capturePlayer   bool
	captureAmount   bool
	duplicatePlayer bool
	duplicateAmount bool
}

// LandTappedForMana matches a land mana ability that used a tap cost and produced mana.
func LandTappedForMana() landTappedForManaPattern { return landTappedForManaPattern{} }

// WithPlayer saves the player who activated the land's mana ability.
func (p landTappedForManaPattern) WithPlayer(binding *PlayerBinding) landTappedForManaPattern {
	if p.capturePlayer {
		p.duplicatePlayer = true
	}
	p.player, p.capturePlayer = binding, true
	return p
}

// WithAmount saves the amount of mana produced by the land ability.
func (p landTappedForManaPattern) WithAmount(binding *NumberBinding) landTappedForManaPattern {
	if p.captureAmount {
		p.duplicateAmount = true
	}
	p.amount, p.captureAmount = binding, true
	return p
}

func (landTappedForManaPattern) eventPattern() {}

func (landTappedForManaPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(EvtTappedForMana, false, effect).SetConditionData(EventSourceWasOfType{Type: TypeLand})
}

func (p landTappedForManaPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "a land is tapped for mana", battlefieldSource: true}
	if p.capturePlayer {
		if p.player == nil || p.duplicatePlayer {
			n.problem = "land tapped for mana: supply one player binding"
		} else {
			n.bindingWrites = append(n.bindingWrites, p.player)
			n.children = append(n.children, definitionEdge{"player", &eventPlayerCapture{binding: p.player}})
		}
	}
	if p.captureAmount {
		if p.amount == nil || p.duplicateAmount {
			n.problem = "land tapped for mana: supply one amount binding"
		} else {
			n.bindingWrites = append(n.bindingWrites, p.amount)
			n.children = append(n.children, definitionEdge{"amount", &eventAmountCapture{binding: p.amount}})
		}
	}
	return n
}

func (p landTappedForManaPattern) captureEventEffects(event *GameEvent, effects []Effect) []Effect {
	if !p.capturePlayer && !p.captureAmount {
		return effects
	}
	var captures []ResolvingEffect
	if p.capturePlayer {
		captures = append(captures, &eventPlayerCapture{binding: p.player, player: event.PlayerID})
	}
	if p.captureAmount {
		captures = append(captures, &eventAmountCapture{binding: p.amount, amount: event.Amount})
	}
	captures = append(captures, Sequence(effects...))
	return []Effect{Sequence(captures...)}
}
