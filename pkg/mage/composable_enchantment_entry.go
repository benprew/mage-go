package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type controlledEnchantmentEntryPattern struct {
	permanent        *PermanentBinding
	capturePermanent bool
	duplicateCapture bool
}

// EnchantmentYouControlEnters matches an enchantment that enters under your control.
func EnchantmentYouControlEnters() controlledEnchantmentEntryPattern {
	return controlledEnchantmentEntryPattern{}
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
	return NewTriggered(EvtZoneChange, false, effect).SetConditionData(AndTriggerCond{Conditions: []TriggerConditionData{
		EventZoneChangeMatches{From: ZoneAny, To: ZoneBattlefield},
		EventSourceHasType{Type: TypeEnchantment},
		EventSourceControlledByController{},
	}})
}

func (p controlledEnchantmentEntryPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "an enchantment you control enters", battlefieldSource: true}
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
