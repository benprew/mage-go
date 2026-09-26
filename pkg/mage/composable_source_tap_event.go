package mage

import . "github.com/benprew/mage-go/pkg/mage/core"

type sourceTappedPattern struct{}

// BecomesTapped matches a battlefield source when its status changes to tapped.
func BecomesTapped(sourceRef) EventPattern { return sourceTappedPattern{} }

func (sourceTappedPattern) eventPattern() {}

func (sourceTappedPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(EvtTapped, false, effect).SetConditionData(EventSourceIsSelf{})
}

func (sourceTappedPattern) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-pattern", Text: "source becomes tapped", battlefieldSource: true}
}
