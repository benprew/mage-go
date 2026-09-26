package mage

import . "github.com/benprew/mage-go/pkg/mage/core"

type sourceCombatEventPattern struct {
	blocking          bool
	unblocked         bool
	defender          *PlayerBinding
	captureDefender   bool
	duplicateDefender bool
}
type sourceBecomesBlockedPattern struct{}

// Attacks matches the source when it is declared as an attacker.
func Attacks(sourceRef) sourceCombatEventPattern { return sourceCombatEventPattern{} }

// AttacksUnblocked matches an attacker after legal blockers are declared.
func AttacksUnblocked(sourceRef) sourceCombatEventPattern {
	return sourceCombatEventPattern{unblocked: true}
}

// WithDefendingPlayer saves the player defending against the source attack.
func (p sourceCombatEventPattern) WithDefendingPlayer(binding *PlayerBinding) sourceCombatEventPattern {
	if p.captureDefender {
		p.duplicateDefender = true
	}
	p.defender = binding
	p.captureDefender = true
	return p
}

// Blocks matches the source once when it is declared as a blocker.
func Blocks(sourceRef) EventPattern { return sourceCombatEventPattern{blocking: true} }

// BecomesBlocked matches the source's first transition to blocked in combat.
func BecomesBlocked(sourceRef) EventPattern { return sourceBecomesBlockedPattern{} }

func (sourceCombatEventPattern) eventPattern()    {}
func (sourceBecomesBlockedPattern) eventPattern() {}

func (sourceBecomesBlockedPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(EvtCreatureBecomesBlocked, false, effect).SetConditionData(EventSourceIsSelf{})
}

func (sourceBecomesBlockedPattern) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-pattern", Text: "source becomes blocked", battlefieldSource: true}
}

func (p sourceCombatEventPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	if p.blocking {
		return BlocksTrigger(effect, false)
	}
	if p.unblocked {
		return NewTriggered(EvtAttacksUnblocked, false, effect).SetConditionData(EventSourceIsSelf{})
	}
	return AttacksTrigger(effect, false)
}

func (p sourceCombatEventPattern) inspectDefinition() definitionNode {
	label := "source attacks"
	if p.blocking {
		label = "source blocks"
	} else if p.unblocked {
		label = "source attacks unblocked"
	}
	n := definitionNode{Role: "event-pattern", Text: label, battlefieldSource: true}
	if p.captureDefender {
		if p.blocking || p.defender == nil || p.duplicateDefender {
			n.problem = "attack event: supply one defending player binding"
		} else {
			n.bindingWrite = p.defender
			n.children = append(n.children, definitionEdge{"defender", &eventPlayerCapture{binding: p.defender}})
		}
	}
	return n
}

func (p sourceCombatEventPattern) captureEventEffects(event *GameEvent, effects []Effect) []Effect {
	if !p.captureDefender {
		return effects
	}
	return []Effect{Sequence(&eventPlayerCapture{binding: p.defender, player: event.TargetID}, Sequence(effects...))}
}
