package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type attachedBecomesTappedPattern struct {
	host *PermanentBinding
}

// AttachedBecomesTapped saves the attached permanent when it becomes tapped.
// Later attachment changes do not change the saved permanent.
func AttachedBecomesTapped(host *PermanentBinding) EventPattern {
	return attachedBecomesTappedPattern{host}
}
func (attachedBecomesTappedPattern) eventPattern() {}
func (attachedBecomesTappedPattern) makeTrigger(body ResolvingEffect) *GenericTriggered {
	return WhenAttachedBecomesTappedTrigger(body, false)
}
func (p attachedBecomesTappedPattern) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-pattern", Text: "attached permanent becomes tapped", battlefieldSource: true,
		children: []definitionEdge{{"host", &eventPermanentCapture{binding: p.host}}}}
}
func (p attachedBecomesTappedPattern) captureDetectedEvent(event *GameEvent, g GameReader, _ uuid.UUID) []Effect {
	var host capturedPermanent
	if perm := g.FindPermanent(event.SourceID); perm != nil {
		host = capturedPermanent{perm.ID(), perm.incarnationID}
	}
	return []Effect{&eventPermanentCapture{p.host, host}}
}
