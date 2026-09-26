package mage

import . "github.com/benprew/mage-go/pkg/mage/core"

type opponentLandTappedPattern struct {
	subtype          string
	filterSubtype    bool
	duplicateSubtype bool
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
	return n
}
