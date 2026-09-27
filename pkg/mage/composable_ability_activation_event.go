package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type abilityActivatedPattern struct {
	activator          *PlayerBinding
	captureActivator   bool
	duplicateActivator bool
	source             *DamageSourceBinding
	captureSource      bool
	duplicateSource    bool
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

// WithSource saves the source as it exists when the ability is activated.
func (p abilityActivatedPattern) WithSource(binding *DamageSourceBinding) abilityActivatedPattern {
	if p.captureSource {
		p.duplicateSource = true
	}
	p.source, p.captureSource = binding, true
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
	if p.captureSource {
		if p.source == nil || p.duplicateSource {
			n.problem = "ability activation event: supply one source binding"
		} else {
			n.bindingWrites = append(n.bindingWrites, p.source)
			n.children = append(n.children, definitionEdge{"source", &eventDamageSourceCapture{binding: p.source}})
		}
	}
	return n
}

func (p abilityActivatedPattern) captureDetectedEvent(event *GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	if !p.captureSource || p.source == nil || event.SourceIncarnation == uuid.Nil {
		return nil
	}
	game, ok := observer.(*Game)
	if !ok {
		return nil
	}
	var source DamageSource
	if permanent := game.FindPermanentIncludingPhased(event.SourceID); permanent != nil && permanent.incarnationID == event.SourceIncarnation {
		source = damageSourceFromPermanent(permanent)
	} else if lki := game.zones.LKIByIncarnation(event.SourceIncarnation); lki != nil && lki.ID == event.SourceID {
		source = damageSourceFromLKI(lki)
	}
	return []Effect{&eventDamageSourceCapture{binding: p.source, source: source}}
}

type eventDamageSourceCapture struct {
	binding *DamageSourceBinding
	source  DamageSource
}

func (*eventDamageSourceCapture) Text() string                 { return "remember the event source" }
func (*eventDamageSourceCapture) Properties() EffectProperties { return EffectProperties{} }
func (e *eventDamageSourceCapture) Apply(ctx *EffectContext) error {
	if ctx.Game.resolution.damageSourceBindings == nil {
		ctx.Game.resolution.damageSourceBindings = make(map[*DamageSourceBinding]DamageSource)
	}
	ctx.Game.resolution.damageSourceBindings[e.binding] = cloneDamageSource(e.source)
	return nil
}
func (e *eventDamageSourceCapture) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-damage-source-capture", bindingWrite: e.binding, children: []definitionEdge{{"source", e.binding}}}
}

func (p abilityActivatedPattern) captureEventEffects(event *GameEvent, effects []Effect) []Effect {
	if !p.captureActivator {
		return effects
	}
	return []Effect{Sequence(&eventPlayerCapture{binding: p.activator, player: event.PlayerID}, Sequence(effects...))}
}
