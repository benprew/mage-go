package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type permanentTappedPattern struct {
	cardType            CardType
	filterType          bool
	duplicateType       bool
	opponentOnly        bool
	duplicateOpponent   bool
	permanent           *PermanentBinding
	capturePermanent    bool
	duplicatePermanent  bool
	controller          *PlayerBinding
	captureController   bool
	duplicateController bool
}

// PermanentBecomesTapped matches a permanent that changes to tapped.
func PermanentBecomesTapped() permanentTappedPattern { return permanentTappedPattern{} }

// OfType limits the event to a permanent with this card type.
func (p permanentTappedPattern) OfType(cardType CardType) permanentTappedPattern {
	if p.filterType {
		p.duplicateType = true
	}
	p.cardType, p.filterType = cardType, true
	return p
}

// ByOpponent limits the event to a permanent an opponent controls.
func (p permanentTappedPattern) ByOpponent() permanentTappedPattern {
	if p.opponentOnly {
		p.duplicateOpponent = true
	}
	p.opponentOnly = true
	return p
}

// WithPermanent saves the tapped permanent's battlefield incarnation.
func (p permanentTappedPattern) WithPermanent(binding *PermanentBinding) permanentTappedPattern {
	if p.capturePermanent {
		p.duplicatePermanent = true
	}
	p.permanent, p.capturePermanent = binding, true
	return p
}

// WithController saves the tapped permanent's controller at the event.
func (p permanentTappedPattern) WithController(binding *PlayerBinding) permanentTappedPattern {
	if p.captureController {
		p.duplicateController = true
	}
	p.controller, p.captureController = binding, true
	return p
}

func (permanentTappedPattern) eventPattern() {}

func (p permanentTappedPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	t := NewTriggered(EvtTapped, false, effect)
	if p.filterType {
		t.SetConditionData(EventSourceHasType{Type: p.cardType})
	}
	if p.opponentOnly {
		t.AndConditionData(EventSourceControlledByOpponent{})
	}
	return t
}

func (p permanentTappedPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "permanent becomes tapped", battlefieldSource: true}
	if p.duplicateType || p.duplicateOpponent || !validTappedPermanentType(p.cardType, p.filterType) {
		n.problem = "permanent tap event: supply each valid filter once"
	}
	if p.capturePermanent {
		if p.permanent == nil || p.duplicatePermanent {
			n.problem = "permanent tap event: supply one permanent binding"
		} else {
			n.children = append(n.children, definitionEdge{"permanent", &eventPermanentCapture{binding: p.permanent}})
			n.bindingWrites = append(n.bindingWrites, p.permanent)
		}
	}
	if p.captureController {
		if p.controller == nil || p.duplicateController {
			n.problem = "permanent tap event: supply one controller binding"
		} else {
			n.children = append(n.children, definitionEdge{"controller", &eventPlayerCapture{binding: p.controller}})
			n.bindingWrites = append(n.bindingWrites, p.controller)
		}
	}
	return n
}

func validTappedPermanentType(cardType CardType, filtered bool) bool {
	if !filtered {
		return true
	}
	switch cardType {
	case TypeCreature, TypeLand, TypeArtifact, TypeEnchantment, TypePlaneswalker:
		return true
	default:
		return false
	}
}

func (p permanentTappedPattern) captureDetectedEvent(event *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	var captures []Effect
	if p.capturePermanent && p.permanent != nil {
		if permanent := observer.FindPermanent(event.SourceID); permanent != nil && !permanent.PhasedOut {
			captures = append(captures, &eventPermanentCapture{binding: p.permanent, value: capturedPermanent{permanent.ID(), permanent.incarnationID}})
		}
	}
	if p.captureController && p.controller != nil {
		captures = append(captures, &eventPlayerCapture{binding: p.controller, player: event.PlayerID})
	}
	return captures
}
