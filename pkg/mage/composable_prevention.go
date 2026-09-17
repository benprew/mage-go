package mage

import "github.com/google/uuid"

// DamageRecipientRef identifies a player or permanent that can receive damage.
type DamageRecipientRef interface {
	damageRecipient()
	resolveRecipient(*EffectContext) (uuid.UUID, bool)
}

func (sourceRef) damageRecipient() {}
func (r sourceRef) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return permanentRecipient(r, ctx)
}
func (savedPermanentRef) damageRecipient() {}
func (r savedPermanentRef) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return permanentRecipient(r, ctx)
}
func (attachedPermanentRef) damageRecipient() {}
func (r attachedPermanentRef) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return permanentRecipient(r, ctx)
}
func (*PermanentTargetBinding) damageRecipient() {}
func (b *PermanentTargetBinding) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return permanentRecipient(b, ctx)
}
func permanentRecipient(r PermanentRef, ctx *EffectContext) (uuid.UUID, bool) {
	p, ok := r.resolvePermanent(ctx)
	if !ok {
		return uuid.Nil, false
	}
	return p.ID(), true
}
func (controllerRef) damageRecipient() {}
func (r controllerRef) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return r.resolvePlayer(ctx)
}
func (activePlayerRef) damageRecipient() {}
func (r activePlayerRef) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return r.resolvePlayer(ctx)
}
func (*PlayerTargetBinding) damageRecipient() {}
func (b *PlayerTargetBinding) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return b.resolvePlayer(ctx)
}

// DamageTargetBinding declares one target player, creature, or planeswalker.
type DamageTargetBinding struct{ *targetDeclaration }

// DeclareDamageTarget declares any damage target. DSL exports it as TargetDamageRecipient.
func DeclareDamageTarget(name string) *DamageTargetBinding {
	return &DamageTargetBinding{&targetDeclaration{name, TargetDamageAnyTarget()}}
}
func (*DamageTargetBinding) damageRecipient() {}
func (b *DamageTargetBinding) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	values := ctx.targetBindings[b.targetDeclaration]
	if len(values) == 0 {
		return uuid.Nil, false
	}
	value, _ := firstTargetValue(values)
	if ctx.Game.GetPlayer(value.id) != nil {
		return value.id, true
	}
	p := ctx.Game.FindPermanent(value.id)
	return value.id, p != nil && !p.PhasedOut && p.incarnationID == value.incarnation
}
func (b *DamageTargetBinding) inspectDefinition() definitionNode {
	return targetReadNode(b.targetDeclaration, "damage-recipient-ref")
}

type preventionInstallation struct {
	recipient DamageRecipientRef
	amount    ValueSource
}

// PreventNextDamage installs an end-of-turn shield in the existing replacement pipeline.
func PreventNextDamage(recipient DamageRecipientRef, amount ValueSource) ResolvingEffect {
	return &preventionInstallation{recipient, amount}
}
func (e *preventionInstallation) Text() string { return "prevent the next damage this turn" }
func (e *preventionInstallation) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *preventionInstallation) Apply(ctx *EffectContext) error {
	id, ok := e.recipient.resolveRecipient(ctx)
	if !ok {
		return nil
	}
	amount := e.amount.Resolve(ctx.Game, ctx.SourceID, ctx.Controller, ctx.Targets)
	ctx.Game.AddPreventionShield(id, amount)
	return nil
}
func (e *preventionInstallation) inspectDefinition() definitionNode {
	return definitionNode{Role: "install-prevention", Text: e.Text(), children: []definitionEdge{{"recipient", e.recipient}, {"amount", e.amount}}}
}

// PreventCombatDamageToAndBy prevents combat damage to and by the subject this turn.
func PreventCombatDamageToAndBy(subject PermanentRef) ResolvingEffect {
	return &combatPreventionInstallation{subject}
}

type combatPreventionInstallation struct{ subject PermanentRef }

func (*combatPreventionInstallation) Text() string {
	return "prevent combat damage to and by permanent this turn"
}
func (*combatPreventionInstallation) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *combatPreventionInstallation) Apply(ctx *EffectContext) error {
	if permanent, ok := e.subject.resolvePermanent(ctx); ok {
		ctx.Game.PreventCombatDamageToAndBy(permanent.ID())
	}
	return nil
}
func (e *combatPreventionInstallation) inspectDefinition() definitionNode {
	return definitionNode{Role: "install-combat-prevention", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
}
