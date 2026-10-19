package mage

import (
	"fmt"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// AcquireStaticAbility gives one saved permanent a layer 6 static rule.
// The rule uses the recipient's live source and controller context.
func AcquireStaticAbility(subject PermanentRef, rule StaticRule) ResolvingEffect {
	return &acquireStaticEffect{subject: subject, rule: rule, duration: core.Indefinite}
}

// AcquireStaticAbilityUntil gives one saved visit an ability for a fixed duration.
func AcquireStaticAbilityUntil(subject PermanentRef, rule StaticRule, duration core.Duration) ResolvingEffect {
	return &acquireStaticEffect{subject: subject, rule: rule, duration: duration}
}

type acquireStaticEffect struct {
	subject  PermanentRef
	rule     StaticRule
	duration core.Duration
}

func (*acquireStaticEffect) Text() string { return "acquire static ability" }
func (*acquireStaticEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *acquireStaticEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "acquire-static", children: []definitionEdge{{"subject", e.subject}, {"ability", acquiredStaticDefinition{e.rule}}}}
	if e.duration != core.Indefinite && e.duration != core.EndOfTurn && e.duration != core.EndOfCombat {
		n.problem = "static ability grant requires a supported duration"
	}
	return n
}

func (e *acquireStaticEffect) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.subject)
}

type acquiredStaticDefinition struct{ rule StaticRule }

func (d acquiredStaticDefinition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "acquired-static-rule", bindingScope: true, children: []definitionEdge{{"rule", d.rule}}}
	if _, ok := d.rule.(cantAttackEachRule); ok {
		return n
	}
	if _, ok := d.rule.(subjectUntapRule); ok {
		return n
	}
	rule, ok := d.rule.(*continuousRule)
	if !ok || rule == nil {
		n.problem = "acquired static ability requires a continuous rule"
		return n
	}
	if nilDefinition(rule.condition) || len(rule.modifications) == 0 {
		n.problem = "acquired static ability requires a condition and modifications"
	}
	for _, value := range rule.modifications {
		if nilDefinition(value) {
			n.problem = "acquired static ability requires non-nil modifications"
			continue
		}
		m := value.modification()
		if m.layer != core.LayerAbility || m.ability != nil || m.triggerGrant {
			n.problem = "acquired static ability requires layer 6 without nested ability grants"
		}
	}
	return n
}

func (e *acquireStaticEffect) Apply(ctx *EffectContext) error {
	if problem := (acquiredStaticDefinition{e.rule}).inspectDefinition().problem; problem != "" {
		return fmt.Errorf("acquire static ability: %s", problem)
	}
	p, ok := e.subject.resolvePermanent(ctx)
	if !ok {
		return nil
	}
	return e.installOn(ctx, p)
}

func (e *acquireStaticEffect) installOn(ctx *EffectContext, p *Permanent) error {
	duration := e.duration
	holder := Static(e.rule)
	grant := &acquiredStaticGrant{recipient: capturedPermanent{p.ID(), p.incarnationID}, holder: holder, duration: duration}
	grant.SetSourceID(p.ID())
	ctx.Game.AddContinuousEffect(grant)
	for _, effect := range holder.Effects {
		effect.SetSourceID(p.ID())
		ctx.Game.AddContinuousEffect(&printedAbilityContinuousEffect{ContinuousEffect: effect, declaration: holder, incarnation: p.incarnationID, acquired: true, expiry: duration})
	}
	ctx.Game.ApplyContinuousEffects()
	return nil
}

type acquiredStaticGrant struct {
	effectSource
	recipient capturedPermanent
	holder    *StaticAbilityHolder
	duration  core.Duration
}

func (*acquiredStaticGrant) Text() string                 { return "grant acquired static rule" }
func (*acquiredStaticGrant) Properties() EffectProperties { return EffectProperties{} }
func (*acquiredStaticGrant) GetLayer() core.Layer         { return core.LayerAbility }
func (e *acquiredStaticGrant) GetDuration() core.Duration { return e.duration }
func (e *acquiredStaticGrant) IsActive(g *Game) bool {
	p := g.FindPermanent(e.recipient.id)
	return p != nil && !p.PhasedOut && p.incarnationID == e.recipient.incarnation
}
func (e *acquiredStaticGrant) ApplyContinuous(ctx *EffectContext) error {
	if !e.IsActive(ctx.Game) {
		return nil
	}
	p := ctx.Game.MutablePermanent(e.recipient.id)
	if !slices.ContainsFunc(p.RuntimeAbilities, func(a Ability) bool { return a.AbilityID() == e.holder.AbilityID() }) {
		p.RuntimeAbilities = append(p.RuntimeAbilities, WrapGrantedAbility(e.holder))
	}
	return nil
}
func (e *acquiredStaticGrant) cloneEffect() ContinuousEffect { cloned := *e; return &cloned }
