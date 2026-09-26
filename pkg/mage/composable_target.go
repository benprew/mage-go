package mage

import (
	"fmt"

	"github.com/google/uuid"
)

// TargetBinding declares a target group. The declaration stores no choices.
type TargetBinding interface {
	targetBinding()
	targetSpec() Target
	targetName() string
}

type targetDeclaration struct {
	name string
	spec Target
}

func (*targetDeclaration) targetBinding()       {}
func (b *targetDeclaration) targetSpec() Target { return b }
func (b *targetDeclaration) targetName() string { return b.name }
func (b *targetDeclaration) Possible(c uuid.UUID, card Card, g *Game) []uuid.UUID {
	return b.spec.Possible(c, card, g)
}
func (*targetDeclaration) Choose(uuid.UUID, Card, *Game, []uuid.UUID) error { return nil }
func (*targetDeclaration) Chosen() []uuid.UUID                              { return nil }
func (*targetDeclaration) IsChosen() bool                                   { return false }
func (*targetDeclaration) Reset()                                           {}
func (b *targetDeclaration) Min() int                                       { return b.spec.Min() }
func (b *targetDeclaration) Max() int                                       { return b.spec.Max() }

// UnwrapTarget returns the underlying Target if wrapped in a targetDeclaration.
func UnwrapTarget(t Target) Target {
	if td, ok := t.(*targetDeclaration); ok {
		return td.spec
	}
	return t
}

// PermanentTargetBinding is a singular permanent target reference.
type PermanentTargetBinding struct{ *targetDeclaration }

// DeclarePermanentTarget declares one permanent target. DSL exports it as TargetPermanent.
func DeclarePermanentTarget(name string, filter PermanentFilter, options ...PermanentTargetOption) *PermanentTargetBinding {
	spec := TargetPermanent(filter).(*PermanentTarget)
	applyPermanentTargetOptions(spec, options)
	return &PermanentTargetBinding{&targetDeclaration{name, spec}}
}
func (*PermanentTargetBinding) permanentRef() {}
func (b *PermanentTargetBinding) resolvePermanent(ctx *EffectContext) (*Permanent, bool) {
	values := ctx.targetBindings[b.targetDeclaration]
	if len(values) == 0 {
		return nil, false
	}
	value, _ := firstTargetValue(values)
	p := ctx.Game.FindPermanent(value.id)
	return p, p != nil && !p.PhasedOut && p.incarnationID == value.incarnation
}
func (b *PermanentTargetBinding) inspectDefinition() definitionNode {
	return targetReadNode(b.targetDeclaration, "permanent-ref")
}

// PlayerTargetBinding is a singular player target reference.
type PlayerTargetBinding struct{ *targetDeclaration }

// DeclarePlayerTarget declares one player target. DSL exports it as TargetPlayer.
func DeclarePlayerTarget(name string, options ...PlayerTargetOption) *PlayerTargetBinding {
	spec := TargetPlayer()
	for _, option := range options {
		if nilDefinition(option) {
			panic("targets: supply a non-nil player target option")
		}
		spec = option.applyPlayerTarget(spec)
	}
	return &PlayerTargetBinding{&targetDeclaration{name, spec}}
}

// PlayerTargetOption adds a restriction to a player target declaration.
type PlayerTargetOption interface {
	playerTargetOption()
	applyPlayerTarget(Target) Target
}

type targetOpponentOnlyOption struct{}

// TargetOpponentOnly restricts the target to an opponent of the ability controller.
func TargetOpponentOnly() PlayerTargetOption                     { return targetOpponentOnlyOption{} }
func (targetOpponentOnlyOption) playerTargetOption()             {}
func (targetOpponentOnlyOption) applyPlayerTarget(Target) Target { return TargetOpponent() }
func (targetOpponentOnlyOption) inspectDefinition() definitionNode {
	return definitionNode{Role: "target-player", Text: "opponent"}
}
func (*PlayerTargetBinding) playerRef() {}
func (b *PlayerTargetBinding) resolvePlayer(ctx *EffectContext) (uuid.UUID, bool) {
	values := ctx.targetBindings[b.targetDeclaration]
	value, ok := firstTargetValue(values)
	return value.id, ok && ctx.Game.GetPlayer(value.id) != nil
}
func (b *PlayerTargetBinding) inspectDefinition() definitionNode {
	return targetReadNode(b.targetDeclaration, "player-ref")
}
func targetReadNode(b *targetDeclaration, role string) definitionNode {
	n := definitionNode{Role: role, targetRead: b}
	if b == nil {
		n.problem = "supply a target binding"
	} else {
		n.Text = b.name
	}
	return n
}

type targetOptions struct{ bindings []TargetBinding }

// WithTargets declares targets for a spell, activation, or trigger.
func WithTargets(bindings ...TargetBinding) targetOptions {
	return targetOptions{append([]TargetBinding(nil), bindings...)}
}
func (targetOptions) spellOption()                          {}
func (targetOptions) activationOption()                     {}
func (o targetOptions) applySpell(a *ActionDefinition)      { applyTargetOptions(a, o.bindings) }
func (o targetOptions) applyActivation(a *ActionDefinition) { applyTargetOptions(a, o.bindings) }
func applyTargetOptions(a *ActionDefinition, bindings []TargetBinding) {
	for _, b := range bindings {
		if nilDefinition(b) {
			panic("targets: supply a non-nil binding")
		}
		a.targets = append(a.targets, b.targetSpec())
	}
}

// SpellOption is a closed option for a resolving spell ability.
type SpellOption interface {
	spellOption()
	applySpell(*ActionDefinition)
}

// ActivationOption is a closed option for an activated ability.
type ActivationOption interface {
	activationOption()
	applyActivation(*ActionDefinition)
}

// Spell creates a spell ability with one effect body.
func Spell(effect ResolvingEffect, options ...SpellOption) *ActionDefinition {
	a := NewAction(ActionSpell, WithEffect(effect))
	for _, o := range options {
		if nilDefinition(o) {
			panic("spell: supply a non-nil option")
		}
		o.applySpell(a)
	}
	return a
}

func declaredTargetNodes(targets []Target, grouped ...bool) []definitionEdge {
	var edges []definitionEdge
	if len(targets) > 0 {
		edges = append(edges, definitionEdge{"target-groups", targetGroupContract{targets: targets, grouped: len(grouped) > 0 && grouped[0]}})
	}
	for i, t := range targets {
		if b, ok := t.(*targetDeclaration); ok {
			edges = append(edges, definitionEdge{fmt.Sprintf("targets[%d]", i), b})
		}
	}
	return edges
}
func (b *targetDeclaration) inspectDefinition() definitionNode {
	n := definitionNode{Role: "target-binding", Text: b.name, targetDeclaration: b}
	if _, ok := b.spec.(*OpponentTarget); ok {
		n.children = append(n.children, definitionEdge{"player", targetOpponentOnlyOption{}})
	}
	if spec, ok := b.spec.(*PermanentTarget); ok && spec.controllerOnly {
		n.children = append(n.children, definitionEdge{"controller", targetYouControlOption{}})
	}
	if spec, ok := b.spec.(*PermanentTarget); ok && spec.opponentOnly {
		n.children = append(n.children, definitionEdge{"controller", targetOpponentControlsOption{}})
	}
	if spec, ok := b.spec.(*PermanentTarget); ok && spec.controllerOwned {
		n.children = append(n.children, definitionEdge{"owner", targetYouOwnOption{}})
	}
	if spec, ok := b.spec.(*PermanentTarget); ok && spec.attachedToControllerOnly {
		n.children = append(n.children, definitionEdge{"attachment-controller", targetAttachedToYouControlOption{}})
	}
	if isOpponentChosenTarget(b.spec) {
		n.children = append(n.children, definitionEdge{"chooser", targetChosenByOpponentOption{}})
	}
	if b.name == "" {
		n.problem = "supply a target binding name"
	}
	return n
}

// CostGroup distinguishes an explicit empty payment from a missing payment.
type CostGroup struct {
	costs   []Cost
	present bool
}

// Costs groups payment instructions for the existing total-cost transaction.
func Costs(costs ...Cost) CostGroup { return CostGroup{append([]Cost(nil), costs...), true} }

// ManaPayment declares a mana payment.
func ManaPayment(value string) Cost { return ManaCostOf(value) }

// Activate creates an activated ability with typed options and a single body.
func Activate(costs CostGroup, effect ResolvingEffect, options ...ActivationOption) *ActionDefinition {
	if !costs.present {
		panic("costs: supply an explicit Costs group")
	}
	a := NewAction(ActionActivated, WithEffect(effect))
	for _, c := range costs.costs {
		if nilDefinition(c) {
			panic("costs: supply a non-nil cost")
		}
		a.costs = append(a.costs, c)
	}
	for _, o := range options {
		if nilDefinition(o) {
			panic("activation: supply a non-nil option")
		}
		o.applyActivation(a)
	}
	return a
}

func firstTargetValue(values []capturedPermanent) (capturedPermanent, bool) {
	if len(values) == 0 {
		return capturedPermanent{}, false
	}
	return values[0], true
}

// PermanentTargetsBinding is a collection target declaration, not a singular reference.
type PermanentTargetsBinding struct{ *targetDeclaration }

// DeclarePermanentTargets declares a bounded collection of permanent targets.
func DeclarePermanentTargets(name string, minimum, maximum int, filter PermanentFilter, options ...PermanentTargetOption) *PermanentTargetsBinding {
	if minimum < 0 || maximum < minimum {
		panic("targets: require 0 <= minimum <= maximum")
	}
	spec := &PermanentTarget{BaseTarget: BaseTarget{min: minimum, max: maximum}, Filters: []PermanentFilter{filter}}
	applyPermanentTargetOptions(spec, options)
	return &PermanentTargetsBinding{&targetDeclaration{name, spec}}
}
func (*PermanentTargetsBinding) permanentSet() {}
func (b *PermanentTargetsBinding) resolvePermanents(ctx *EffectContext) []*Permanent {
	values := ctx.targetBindings[b.targetDeclaration]
	result := make([]*Permanent, 0, len(values))
	for _, value := range values {
		p := ctx.Game.FindPermanent(value.id)
		if p != nil && !p.PhasedOut && p.incarnationID == value.incarnation {
			result = append(result, p)
		}
	}
	return result
}
func (b *PermanentTargetsBinding) inspectDefinition() definitionNode {
	return targetReadNode(b.targetDeclaration, "permanent-set")
}

type targetGroupContract struct {
	targets []Target
	grouped bool
}

func (c targetGroupContract) inspectDefinition() definitionNode {
	n := definitionNode{Role: "target-groups"}
	variable, typed := 0, false
	for _, t := range c.targets {
		if _, ok := t.(*targetDeclaration); ok {
			typed = true
		}
		if t != nil && t.Min() != t.Max() {
			variable++
		}
	}
	if typed && variable > 1 && !c.grouped {
		n.problem = "multiple variable target groups require recorded selection boundaries"
	}
	return n
}

// PermanentTargetOption adds a restriction to a permanent target declaration.
type PermanentTargetOption interface {
	permanentTargetOption()
	applyPermanentTarget(*PermanentTarget)
}

type targetYouControlOption struct{}

// TargetYouControl restricts targets to permanents controlled by the ability controller.
func TargetYouControl() PermanentTargetOption                          { return targetYouControlOption{} }
func (targetYouControlOption) permanentTargetOption()                  {}
func (targetYouControlOption) applyPermanentTarget(t *PermanentTarget) { t.controllerOnly = true }
func applyPermanentTargetOptions(t *PermanentTarget, options []PermanentTargetOption) {
	for _, option := range options {
		if nilDefinition(option) {
			panic("targets: supply a non-nil permanent target option")
		}
		option.applyPermanentTarget(t)
	}
	if t.controllerOnly && t.opponentOnly {
		panic("targets: conflicting controller restrictions")
	}
}

func (targetYouControlOption) inspectDefinition() definitionNode {
	return definitionNode{Role: "target-controller", Text: "you control"}
}

type targetOpponentControlsOption struct{}

// TargetOpponentControls restricts targets to permanents an opponent controls.
func TargetOpponentControls() PermanentTargetOption                          { return targetOpponentControlsOption{} }
func (targetOpponentControlsOption) permanentTargetOption()                  {}
func (targetOpponentControlsOption) applyPermanentTarget(t *PermanentTarget) { t.opponentOnly = true }
func (targetOpponentControlsOption) inspectDefinition() definitionNode {
	return definitionNode{Role: "target-controller", Text: "an opponent controls"}
}

type targetChosenByOpponentOption struct{}

// TargetChosenByOpponent lets the opponent choose the targets at stack placement.
func TargetChosenByOpponent() PermanentTargetOption         { return targetChosenByOpponentOption{} }
func (targetChosenByOpponentOption) permanentTargetOption() {}
func (targetChosenByOpponentOption) applyPermanentTarget(t *PermanentTarget) {
	t.opponentChooses = true
}
func (targetChosenByOpponentOption) inspectDefinition() definitionNode {
	return definitionNode{Role: "target-chooser", Text: "opponent"}
}

// OpponentChoosesTarget reports which player must choose the targets.
func (t *PermanentTarget) OpponentChoosesTarget() bool { return t.opponentChooses }

// OpponentChoosesTarget keeps the chooser of the enclosed target specification.
func (b *targetDeclaration) OpponentChoosesTarget() bool { return isOpponentChosenTarget(b.spec) }

type targetYouOwnOption struct{}

// TargetYouOwn restricts targets to permanents owned by the ability controller.
func TargetYouOwn() PermanentTargetOption                          { return targetYouOwnOption{} }
func (targetYouOwnOption) permanentTargetOption()                  {}
func (targetYouOwnOption) applyPermanentTarget(t *PermanentTarget) { t.controllerOwned = true }
func (targetYouOwnOption) inspectDefinition() definitionNode {
	return definitionNode{Role: "target-owner", Text: "you own"}
}

// TargetOtherThanSource excludes the source card from the target choices.
func TargetOtherThanSource() PermanentTargetOption { return targetOtherThanSourceOption{} }

type targetOtherThanSourceOption struct{}

func (targetOtherThanSourceOption) permanentTargetOption()                  {}
func (targetOtherThanSourceOption) applyPermanentTarget(t *PermanentTarget) { t.excludeSource = true }
func (targetOtherThanSourceOption) inspectDefinition() definitionNode {
	return definitionNode{Role: "target-exclude-source", Text: "other than this permanent"}
}

type targetAttachedToYouControlOption struct{}

// TargetAttachedToYouControl restricts targets to attachments on permanents you control.
func TargetAttachedToYouControl() PermanentTargetOption         { return targetAttachedToYouControlOption{} }
func (targetAttachedToYouControlOption) permanentTargetOption() {}
func (targetAttachedToYouControlOption) applyPermanentTarget(t *PermanentTarget) {
	t.attachedToControllerOnly = true
}
func (targetAttachedToYouControlOption) inspectDefinition() definitionNode {
	return definitionNode{Role: "target-attachment-controller", Text: "you control"}
}
