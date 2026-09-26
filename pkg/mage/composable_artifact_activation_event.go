package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type artifactActivationPattern struct {
	artifact            *PermanentBinding
	controller          *PlayerBinding
	captureArtifact     bool
	captureController   bool
	duplicateArtifact   bool
	duplicateController bool
	attachedOnly        bool
}

// ArtifactAbilityActivatedWithoutTapCost matches an artifact ability with no tap symbol in its cost.
func ArtifactAbilityActivatedWithoutTapCost() artifactActivationPattern {
	return artifactActivationPattern{}
}

// AttachedToSource limits the event to the artifact enchanted by the source.
func (p artifactActivationPattern) AttachedToSource() artifactActivationPattern {
	p.attachedOnly = true
	return p
}

// WithArtifact saves the activated artifact's battlefield incarnation.
func (p artifactActivationPattern) WithArtifact(binding *PermanentBinding) artifactActivationPattern {
	if p.captureArtifact {
		p.duplicateArtifact = true
	}
	p.artifact, p.captureArtifact = binding, true
	return p
}

// WithController saves the artifact controller when the ability is activated.
func (p artifactActivationPattern) WithController(binding *PlayerBinding) artifactActivationPattern {
	if p.captureController {
		p.duplicateController = true
	}
	p.controller, p.captureController = binding, true
	return p
}

func (artifactActivationPattern) eventPattern() {}

func (p artifactActivationPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	t := NewTriggered(EvtAbilityActivated, false, effect).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
		activationWithoutTapCost{}, EventSourceHasType{Type: TypeArtifact},
	}})
	if p.attachedOnly {
		t.AndConditionData(AttachedToIsEventSourceNoTapCost{})
	}
	return t
}

type activationWithoutTapCost struct{}

func (activationWithoutTapCost) CheckTriggerCond(event *GameEvent, _ GameReader, _, _ uuid.UUID) bool {
	return !event.Flag
}

func (p artifactActivationPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "artifact ability activated without tap cost", battlefieldSource: true}
	if p.attachedOnly {
		n.Text = "attached artifact ability activated without tap cost"
	}
	if p.captureArtifact {
		if p.artifact == nil || p.duplicateArtifact {
			n.problem = "artifact activation event: supply one artifact binding"
		} else {
			n.children = append(n.children, definitionEdge{"artifact", &eventPermanentCapture{binding: p.artifact}})
			n.bindingWrites = append(n.bindingWrites, p.artifact)
		}
	}
	if p.captureController {
		if p.controller == nil || p.duplicateController {
			n.problem = "artifact activation event: supply one controller binding"
		} else {
			n.children = append(n.children, definitionEdge{"controller", &eventPlayerCapture{binding: p.controller}})
			n.bindingWrites = append(n.bindingWrites, p.controller)
		}
	}
	return n
}

func (p artifactActivationPattern) captureDetectedEvent(event *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	var captures []Effect
	if p.captureArtifact && p.artifact != nil {
		if artifact := observer.FindPermanent(event.SourceID); artifact != nil && !artifact.PhasedOut {
			captures = append(captures, &eventPermanentCapture{binding: p.artifact, value: capturedPermanent{artifact.ID(), artifact.incarnationID}})
		}
	}
	if p.captureController && p.controller != nil {
		if artifact := observer.FindPermanent(event.SourceID); artifact != nil {
			captures = append(captures, &eventPlayerCapture{binding: p.controller, player: artifact.ControllerID()})
		} else if game, ok := observer.(*Game); ok {
			if lki := game.LKI(event.SourceID); lki != nil {
				captures = append(captures, &eventPlayerCapture{binding: p.controller, player: lki.ViewController()})
			}
		}
	}
	return captures
}
