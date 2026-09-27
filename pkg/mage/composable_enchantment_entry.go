package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type controlledEnchantmentEntryPattern struct {
	subtype          string
	filterSubtype    bool
	duplicateSubtype bool
	permanent        *PermanentBinding
	capturePermanent bool
	duplicateCapture bool
}

// EnchantmentYouControlEnters matches an enchantment that enters under your control.
func EnchantmentYouControlEnters() controlledEnchantmentEntryPattern {
	return controlledEnchantmentEntryPattern{}
}

// OfSubtype checks the entering enchantment's effective subtype.
func (p controlledEnchantmentEntryPattern) OfSubtype(subtype string) controlledEnchantmentEntryPattern {
	if p.filterSubtype {
		p.duplicateSubtype = true
	}
	p.subtype, p.filterSubtype = subtype, true
	return p
}

// WithPermanent saves the entering enchantment's battlefield incarnation.
func (p controlledEnchantmentEntryPattern) WithPermanent(permanent *PermanentBinding) controlledEnchantmentEntryPattern {
	if p.capturePermanent {
		p.duplicateCapture = true
	}
	p.permanent, p.capturePermanent = permanent, true
	return p
}

func (controlledEnchantmentEntryPattern) eventPattern() {}

func (p controlledEnchantmentEntryPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	t := NewTriggered(EvtZoneChange, false, effect).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
		EventZoneChangeMatches{From: ZoneAny, To: ZoneBattlefield},
		EventSourceHasType{Type: TypeEnchantment},
		EventSourceControlledByController{},
	}})
	if p.filterSubtype {
		t.AndConditionData(eventSourceHasEntrySubtype{subtype: p.subtype})
	}
	return t
}

func (p controlledEnchantmentEntryPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "an enchantment you control enters", battlefieldSource: true}
	if p.filterSubtype {
		if family, known := knownSubtypeFamily(p.subtype); !known || family != SubtypeEnchantment || p.duplicateSubtype {
			n.problem = "enchantment entry event: supply one enchantment subtype"
		} else {
			n.Text = "an enchantment you control enters with subtype " + p.subtype
		}
	}
	if p.capturePermanent {
		if p.permanent == nil || p.duplicateCapture {
			n.problem = "enchantment entry event: supply one permanent binding"
		} else {
			n.bindingWrite = p.permanent
			n.children = append(n.children, definitionEdge{"permanent", &eventPermanentCapture{binding: p.permanent}})
		}
	}
	return n
}

func (p controlledEnchantmentEntryPattern) captureDetectedEvent(event *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	if !p.capturePermanent || p.permanent == nil {
		return nil
	}
	permanent := observer.FindPermanent(event.SourceID)
	if permanent == nil {
		return nil
	}
	return []Effect{&eventPermanentCapture{binding: p.permanent, value: capturedPermanent{permanent.ID(), permanent.incarnationID}}}
}
