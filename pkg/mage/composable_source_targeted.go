package mage

import . "github.com/benprew/mage-go/pkg/mage/core"

type sourceTargetedBySpellPattern struct{}

// BecomesTargetOfSpell matches a spell that targets the battlefield source.
func BecomesTargetOfSpell(sourceRef) EventPattern { return sourceTargetedBySpellPattern{} }

func (sourceTargetedBySpellPattern) eventPattern() {}

func (sourceTargetedBySpellPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(EvtBecomesTarget, false, effect).SetConditionData(AndTriggerCond{
		Conditions: []TriggerConditionData{EventTargetIsSelf{}, EventFlagIsFalse{}},
	})
}

func (sourceTargetedBySpellPattern) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-pattern", Text: "source becomes the target of a spell", battlefieldSource: true}
}
