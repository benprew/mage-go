package mage

import . "github.com/benprew/mage-go/pkg/mage/core"

type sourceCombatEventPattern struct{ blocking bool }
type sourceBecomesBlockedPattern struct{}

// Attacks matches the source when it is declared as an attacker.
func Attacks(sourceRef) EventPattern { return sourceCombatEventPattern{} }

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
	return AttacksTrigger(effect, false)
}

func (p sourceCombatEventPattern) inspectDefinition() definitionNode {
	label := "source attacks"
	if p.blocking {
		label = "source blocks"
	}
	return definitionNode{Role: "event-pattern", Text: label, battlefieldSource: true}
}
