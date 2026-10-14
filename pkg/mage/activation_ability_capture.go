package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// AbilityBinding saves one activated ability without a target declaration.
type AbilityBinding struct{ stack *SpellBinding }

// BindAbility declares an event ability binding.
func BindAbility(name string) *AbilityBinding { return &AbilityBinding{stack: BindSpell(name)} }

func (b *AbilityBinding) bindingName() string {
	if b.stack == nil {
		return ""
	}
	return b.stack.name
}
func (b *AbilityBinding) inspectDefinition() definitionNode {
	n := definitionNode{Role: "ability-binding", Text: b.bindingName(), bindingDeclaration: b}
	if b.bindingName() == "" {
		n.problem = "supply a binding name"
	}
	return n
}

type savedAbilityRef struct{ binding *AbilityBinding }

// FromAbility reads the exact stack ability saved by an event.
func FromAbility(binding *AbilityBinding) AbilityRef { return savedAbilityRef{binding} }

func (savedAbilityRef) abilityRef()     {}
func (savedAbilityRef) stackObjectRef() {}
func (r savedAbilityRef) resolveStackObject(ctx *EffectContext) (*StackObject, bool) {
	if r.binding == nil || r.binding.stack == nil {
		return nil, false
	}
	obj := ctx.Game.stack.FindByID(ctx.Game.resolution.spellBindings[r.binding.stack])
	return obj, obj != nil && obj.IsAbility && obj.activatedAbility
}
func (r savedAbilityRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "ability-ref", Text: "event ability", bindingRead: r.binding, children: []definitionEdge{{"binding", r.binding}}}
}

// FromAttachedPermanent limits activation events to the current attachment.
func (p abilityActivatedPattern) FromAttachedPermanent() abilityActivatedPattern {
	if p.attachedOnly {
		p.duplicateAttached = true
	}
	p.attachedOnly = true
	return p
}

// WithTapCost requires a tap symbol in the activation cost.
func (p abilityActivatedPattern) WithTapCost() abilityActivatedPattern {
	if p.withTapCost {
		p.duplicateTapFilter = true
	}
	p.withTapCost = true
	return p
}

// NonMana limits activation events to abilities that use the stack.
func (p abilityActivatedPattern) NonMana() abilityActivatedPattern {
	if p.nonMana {
		p.duplicateNonMana = true
	}
	p.nonMana = true
	return p
}

// WithAbility saves the activated ability's stack identity.
func (p abilityActivatedPattern) WithAbility(binding *AbilityBinding) abilityActivatedPattern {
	if p.captureAbility {
		p.duplicateAbility = true
	}
	p.ability, p.captureAbility = binding, true
	return p
}

type activationFromAttachedPermanent struct{}

func (activationFromAttachedPermanent) CheckTriggerCond(event *core.GameEvent, g GameReader, source, _ uuid.UUID) bool {
	aura := g.FindPermanent(source)
	host := g.FindPermanent(event.SourceID)
	return aura != nil && host != nil && !host.PhasedOut && aura.AttachedTo == host.ID() && aura.attachedIncarnation == host.incarnationID &&
		event.SourceIncarnation != uuid.Nil && event.SourceIncarnation == host.incarnationID
}

type activationHasTapCost struct{}

func (activationHasTapCost) CheckTriggerCond(event *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
	return event.Flag
}

type activationUsesStack struct{}

func (activationUsesStack) CheckTriggerCond(event *core.GameEvent, _ GameReader, _, _ uuid.UUID) bool {
	return event.ActivatedAbilityID != uuid.Nil
}

type eventAbilityCapture struct {
	binding *AbilityBinding
	id      uuid.UUID
}

func (*eventAbilityCapture) Text() string                 { return "remember activated ability" }
func (*eventAbilityCapture) Properties() EffectProperties { return EffectProperties{} }
func (e *eventAbilityCapture) Apply(ctx *EffectContext) error {
	return (&eventSpellCapture{binding: e.binding.stack, id: e.id}).Apply(ctx)
}
func (e *eventAbilityCapture) inspectDefinition() definitionNode {
	return definitionNode{Role: "event-ability-capture", bindingWrite: e.binding, children: []definitionEdge{{"ability", e.binding}}}
}
