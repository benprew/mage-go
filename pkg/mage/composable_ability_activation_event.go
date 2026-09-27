package mage

import . "github.com/benprew/mage-go/pkg/mage/core"

type abilityActivatedPattern struct {
	activator          *PlayerBinding
	captureActivator   bool
	duplicateActivator bool
}

// ActivatedAbilityEvent matches a completed activated ability announcement.
func ActivatedAbilityEvent() abilityActivatedPattern { return abilityActivatedPattern{} }

// WithActivator saves the player who activated the ability.
func (p abilityActivatedPattern) WithActivator(binding *PlayerBinding) abilityActivatedPattern {
	if p.captureActivator {
		p.duplicateActivator = true
	}
	p.activator, p.captureActivator = binding, true
	return p
}

func (abilityActivatedPattern) eventPattern() {}

func (abilityActivatedPattern) makeTrigger(effect ResolvingEffect) *GenericTriggered {
	return NewTriggered(EvtAbilityActivated, false, effect)
}

func (p abilityActivatedPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "an ability is activated", battlefieldSource: true}
	if p.captureActivator {
		if p.activator == nil || p.duplicateActivator {
			n.problem = "ability activation event: supply one activator binding"
		} else {
			n.bindingWrite = p.activator
			n.children = append(n.children, definitionEdge{"activator", &eventPlayerCapture{binding: p.activator}})
		}
	}
	return n
}

func (p abilityActivatedPattern) captureEventEffects(event *GameEvent, effects []Effect) []Effect {
	if !p.captureActivator {
		return effects
	}
	return []Effect{Sequence(&eventPlayerCapture{binding: p.activator, player: event.PlayerID}, Sequence(effects...))}
}
