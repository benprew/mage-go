package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type opponentLandTappedPattern struct {
	subtype          string
	filterSubtype    bool
	duplicateSubtype bool
	land             *PermanentBinding
	captureLand      bool
	duplicateLand    bool
}

// OpponentLandBecomesTapped matches a land an opponent controls when it becomes tapped.
func OpponentLandBecomesTapped() opponentLandTappedPattern { return opponentLandTappedPattern{} }

// OfSubtype limits the event to a land with the given subtype.
func (p opponentLandTappedPattern) OfSubtype(subtype string) opponentLandTappedPattern {
	if p.filterSubtype {
		p.duplicateSubtype = true
	}
	p.subtype = subtype
	p.filterSubtype = true
	return p
}

// WithLand saves the tapped land's battlefield incarnation for the trigger.
func (p opponentLandTappedPattern) WithLand(binding *PermanentBinding) opponentLandTappedPattern {
	if p.captureLand {
		p.duplicateLand = true
	}
	p.land, p.captureLand = binding, true
	return p
}

func (opponentLandTappedPattern) eventPattern() {}

func (p opponentLandTappedPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	t := NewTriggered(EvtTapped, false, effect).SetConditionData(EventSourceHasType{Type: TypeLand})
	t.AndConditionData(EventSourceControlledByOpponent{})
	if p.filterSubtype && p.subtype != "" {
		t.AndConditionData(EventSourceHasSubType{SubType: p.subtype})
	}
	return t
}

func (p opponentLandTappedPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "opponent land becomes tapped", battlefieldSource: true}
	if p.filterSubtype {
		n.Text = "opponent " + p.subtype + " becomes tapped"
		if p.duplicateSubtype || !isLandSubtype(p.subtype) {
			n.problem = "land tap event: supply a valid land subtype"
		}
	}
	if p.captureLand {
		if p.land == nil || p.duplicateLand {
			n.problem = "land tap event: supply one land binding"
		} else {
			n.children = append(n.children, definitionEdge{"land", &eventPermanentCapture{binding: p.land}})
			n.bindingWrites = append(n.bindingWrites, p.land)
		}
	}
	return n
}

func (p opponentLandTappedPattern) captureDetectedEvent(event *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	if !p.captureLand || p.land == nil {
		return nil
	}
	land := observer.FindPermanent(event.SourceID)
	if land == nil || land.PhasedOut {
		return nil
	}
	return []Effect{&eventPermanentCapture{binding: p.land, value: capturedPermanent{land.ID(), land.incarnationID}}}
}
