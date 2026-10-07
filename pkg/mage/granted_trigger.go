package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type grantedTriggerRule struct {
	subjects PermanentSet
	ability  *GenericTriggered
}

// GrantTriggeredAbility uses the recipient as the trigger source.
// The subject is read in the donor's context.
func GrantTriggeredAbility(subject PermanentRef, ability *GenericTriggered) StaticRule {
	return GrantEachTriggeredAbility(grantRecipientSet{subject}, ability)
}

// GrantEachTriggeredAbility gives each current member a separate trigger.
// Event and action bindings belong to the recipient's ability.
func GrantEachTriggeredAbility(subjects PermanentSet, ability *GenericTriggered) StaticRule {
	return &grantedTriggerRule{subjects: subjects, ability: ability}
}

func (*grantedTriggerRule) staticRule() {}
func (r *grantedTriggerRule) staticEffects() []ContinuousEffect {
	return []ContinuousEffect{&grantedTriggerEffect{rule: r, grantID: uuid.New()}}
}
func (r *grantedTriggerRule) inspectDefinition() definitionNode {
	n := definitionNode{Role: "static-trigger-grant", battlefieldSource: true, children: []definitionEdge{{"subjects", r.subjects}}}
	if r.ability == nil {
		n.problem = "grant requires a triggered ability"
	} else if r.ability.eventDefinition == nil && r.ability.stepDefinition == nil {
		n.problem = "grant requires a declared event or step pattern"
	} else if r.ability.sourceZone != core.ZoneBattlefield || len(r.ability.zones) != 0 {
		n.problem = "grant requires a battlefield trigger"
	} else {
		n.children = append(n.children, definitionEdge{"ability", grantedTriggerDefinition{r.ability}})
	}
	return n
}

type grantedTriggerDefinition struct{ ability *GenericTriggered }

func (d grantedTriggerDefinition) inspectDefinition() definitionNode {
	return definitionNode{Role: "granted-trigger", bindingScope: true, children: []definitionEdge{{"definition", d.ability}}}
}

type grantedTriggerEffect struct {
	effectSource
	rule    *grantedTriggerRule
	grantID uuid.UUID
}

func (*grantedTriggerEffect) Text() string                 { return "grant triggered ability" }
func (*grantedTriggerEffect) Properties() EffectProperties { return EffectProperties{} }
func (*grantedTriggerEffect) GetLayer() core.Layer         { return core.LayerAbility }
func (*grantedTriggerEffect) GetDuration() core.Duration   { return core.WhileOnBattlefield }
func (e *grantedTriggerEffect) IsActive(g *Game) bool {
	p := g.FindPermanent(e.sourceID)
	return p != nil && !p.PhasedOut
}
func (e *grantedTriggerEffect) ApplyContinuous(ctx *EffectContext) error {
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
		var identity [32]byte
		copy(identity[:16], source.incarnationID[:])
		copy(identity[16:], p.incarnationID[:])
		ability := *e.rule.ability
		ability.id = uuid.NewSHA1(e.grantID, identity[:])
		ability.source, ability.controller = p.ID(), p.ControllerID()
		p.RuntimeAbilities = append(p.RuntimeAbilities, WrapGrantedAbility(&ability))
	}
	return nil
}
func (e *grantedTriggerEffect) cloneEffect() ContinuousEffect { clone := *e; return &clone }
func (e *grantedTriggerEffect) inspectDefinition() definitionNode {
	return e.rule.inspectDefinition()
}
