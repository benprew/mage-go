package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type excludedPermanentSet struct {
	subjects PermanentSet
	excluded PermanentRef
}

// WithoutPermanent removes one permanent reference from a set.
func WithoutPermanent(subjects PermanentSet, excluded PermanentRef) PermanentSet {
	return excludedPermanentSet{subjects, excluded}
}
func (excludedPermanentSet) permanentSet() {}
func (s excludedPermanentSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	excluded, ok := s.excluded.resolvePermanent(ctx)
	var result []*Permanent
	for _, p := range s.subjects.resolvePermanents(ctx) {
		if !ok || p.ID() != excluded.ID() {
			result = append(result, p)
		}
	}
	return result
}
func (s excludedPermanentSet) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-set", Text: "exclude permanent", children: []definitionEdge{{"subjects", s.subjects}, {"excluded", s.excluded}}}
}

type grantedActivationRule struct {
	subjects PermanentSet
	ability  *ActionDefinition
}

type grantRecipientSet struct{ subject PermanentRef }

func (grantRecipientSet) permanentSet() {}
func (s grantRecipientSet) resolvePermanents(ctx *EffectContext) []*Permanent {
	if p, ok := s.subject.resolvePermanent(ctx); ok {
		return []*Permanent{p}
	}
	return nil
}
func (s grantRecipientSet) inspectDefinition() definitionNode {
	return definitionNode{Role: "permanent-set", children: []definitionEdge{{"subject", s.subject}}}
}

func grantRecipient(subject PermanentRef) PermanentSet {
	if nilDefinition(subject) {
		panic("grant: supply a permanent reference")
	}
	return grantRecipientSet{subject}
}

// GrantActivatedAbility gives one current permanent an activated ability.
// The subject uses the donor context. The body uses the recipient context.
func GrantActivatedAbility(subject PermanentRef, costs CostGroup, body ResolvingEffect, options ...GrantedActivationOption) StaticRule {
	return GrantEachActivatedAbility(grantRecipient(subject), costs, body, options...)
}

// GrantActivatedModes gives one current permanent an ability with a mode choice.
// Each mode declares its own targets.
func GrantActivatedModes(subject PermanentRef, costs CostGroup, options ...ModalActivationOption) StaticRule {
	return GrantEachActivatedModes(grantRecipient(subject), costs, options...)
}

// GrantedActivationOption is a closed option for a static activation grant.
type GrantedActivationOption interface {
	grantedActivationOption()
	applyGrantedActivation(*ActionDefinition)
}

func (targetOptions) grantedActivationOption() {}
func (o targetOptions) applyGrantedActivation(a *ActionDefinition) {
	applyTargetOptions(a, o.bindings)
}

// GrantEachActivatedAbility gives each set member an activated ability.
// The set is evaluated in layer 6. The body uses the recipient as its source.
func GrantEachActivatedAbility(subjects PermanentSet, costs CostGroup, body ResolvingEffect, options ...GrantedActivationOption) StaticRule {
	if nilDefinition(subjects) {
		panic("grant: supply a permanent set")
	}
	ability := Activate(costs, body)
	for _, option := range options {
		if nilDefinition(option) {
			panic("grant: supply a non-nil option")
		}
		option.applyGrantedActivation(ability)
	}
	return &grantedActivationRule{subjects: subjects, ability: ability}
}

// GrantEachActivatedModes gives each set member an ability that chooses one mode.
// Each mode declares its own targets. Costs are paid by the recipient's controller.
func GrantEachActivatedModes(subjects PermanentSet, costs CostGroup, options ...ModalActivationOption) StaticRule {
	if nilDefinition(subjects) {
		panic("grant: supply a permanent set")
	}
	return &grantedActivationRule{subjects: subjects, ability: ActivateModes(costs, options...)}
}
func (*grantedActivationRule) staticRule() {}
func (r *grantedActivationRule) staticEffects() []ContinuousEffect {
	return []ContinuousEffect{&grantedActivationEffect{rule: r, grantID: uuid.New()}}
}
func (r *grantedActivationRule) inspectDefinition() definitionNode {
	return definitionNode{Role: "static-ability-grant", children: []definitionEdge{{"subjects", r.subjects}, {"ability", grantedActivationDefinition{r.ability}}}}
}

type grantedActivationDefinition struct{ ability *ActionDefinition }

func (d grantedActivationDefinition) inspectDefinition() definitionNode {
	return definitionNode{Role: "granted-activation", bindingScope: true, children: []definitionEdge{{"definition", d.ability}}}
}

type grantedActivationEffect struct {
	effectSource
	rule    *grantedActivationRule
	grantID uuid.UUID
}

func (*grantedActivationEffect) Text() string                 { return "grant activated ability" }
func (*grantedActivationEffect) Properties() EffectProperties { return EffectProperties{} }
func (*grantedActivationEffect) GetLayer() Layer              { return LayerAbility }
func (*grantedActivationEffect) GetDuration() Duration        { return WhileOnBattlefield }
func (e *grantedActivationEffect) IsActive(g *Game) bool {
	p := g.FindPermanent(e.sourceID)
	return p != nil && !p.PhasedOut
}
func (e *grantedActivationEffect) ApplyContinuous(ctx *EffectContext) error {
	source := ctx.Game.FindPermanent(e.sourceID)
	if source == nil || source.PhasedOut {
		return nil
	}
	child := *ctx
	child.SourceID, child.Controller = source.ID(), source.ControllerID()
	child.sourceCaptured, child.sourceIncarnation = true, source.incarnationID
	for _, member := range e.rule.subjects.resolvePermanents(&child) {
		p := ctx.Game.MutablePermanent(member.ID())
		if p == nil || p.PhasedOut {
			continue
		}
		// A grant keeps its limit through layer evaluations and control changes.
		var identity [32]byte
		copy(identity[:16], source.incarnationID[:])
		copy(identity[16:], p.incarnationID[:])
		installGrantedActivation(p, e.rule.ability, uuid.NewSHA1(e.grantID, identity[:]))
	}
	return nil
}
func (e *grantedActivationEffect) cloneEffect() ContinuousEffect { cloned := *e; return &cloned }
func (e *grantedActivationEffect) inspectDefinition() definitionNode {
	return e.rule.inspectDefinition()
}

func installGrantedActivation(p *Permanent, definition *ActionDefinition, id uuid.UUID) {
	ability := *definition
	ability.id = id
	ability.staticGrant = true
	ability.source, ability.controller = p.ID(), p.ControllerID()
	p.RuntimeAbilities = append(p.RuntimeAbilities, WrapGrantedAbility(&ability))
}
