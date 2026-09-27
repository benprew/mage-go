package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type artifactTappedPattern struct {
	artifact          *PermanentBinding
	controller        *PlayerBinding
	captureArtifact   bool
	captureController bool
	duplicateArtifact bool
	duplicatePlayer   bool
	opponentOnly      bool
	duplicateOpponent bool
}

// ArtifactBecomesTapped matches an artifact that changes from untapped to tapped.
func ArtifactBecomesTapped() artifactTappedPattern { return artifactTappedPattern{} }

// ByOpponent limits the event to an artifact an opponent controls when it taps.
func (p artifactTappedPattern) ByOpponent() artifactTappedPattern {
	if p.opponentOnly {
		p.duplicateOpponent = true
	}
	p.opponentOnly = true
	return p
}

// WithArtifact saves the tapped artifact's battlefield incarnation.
func (p artifactTappedPattern) WithArtifact(binding *PermanentBinding) artifactTappedPattern {
	if p.captureArtifact {
		p.duplicateArtifact = true
	}
	p.artifact, p.captureArtifact = binding, true
	return p
}

// WithController saves the tapped artifact's controller at the tap event.
func (p artifactTappedPattern) WithController(binding *PlayerBinding) artifactTappedPattern {
	if p.captureController {
		p.duplicatePlayer = true
	}
	p.controller, p.captureController = binding, true
	return p
}

func (artifactTappedPattern) eventPattern() {}

func (p artifactTappedPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	t := NewTriggered(EvtTapped, false, effect).SetConditionData(EventSourceHasType{Type: TypeArtifact})
	if p.opponentOnly {
		t.AndConditionData(EventSourceControlledByOpponent{})
	}
	return t
}

func (p artifactTappedPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "artifact becomes tapped", battlefieldSource: true}
	if p.opponentOnly {
		n.Text = "opponent artifact becomes tapped"
	}
	if p.duplicateOpponent {
		n.problem = "artifact tap event: supply each filter once"
	}
	if p.captureArtifact {
		if p.artifact == nil || p.duplicateArtifact {
			n.problem = "artifact tap event: supply one artifact binding"
		} else {
			n.children = append(n.children, definitionEdge{"artifact", &eventPermanentCapture{binding: p.artifact}})
			n.bindingWrites = append(n.bindingWrites, p.artifact)
		}
	}
	if p.captureController {
		if p.controller == nil || p.duplicatePlayer {
			n.problem = "artifact tap event: supply one controller binding"
		} else {
			n.children = append(n.children, definitionEdge{"controller", &eventPlayerCapture{binding: p.controller}})
			n.bindingWrites = append(n.bindingWrites, p.controller)
		}
	}
	return n
}

func (p artifactTappedPattern) captureDetectedEvent(event *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	var captures []Effect
	if p.captureArtifact && p.artifact != nil {
		if artifact := observer.FindPermanent(event.SourceID); artifact != nil && !artifact.PhasedOut {
			captures = append(captures, &eventPermanentCapture{binding: p.artifact, value: capturedPermanent{artifact.ID(), artifact.incarnationID}})
		}
	}
	if p.captureController && p.controller != nil {
		captures = append(captures, &eventPlayerCapture{binding: p.controller, player: event.PlayerID})
	}
	return captures
}
