package mage

import (
	"github.com/benprew/mage-go/pkg/mage/core"
)

// PreventUntapWhileSourceTapped saves an untap-step restriction for one visit.
// Source untap, departure or phasing ends the duration permanently.
func PreventUntapWhileSourceTapped(subject PermanentRef) ResolvingEffect {
	return &tappedUntapRuleEffect{subject: subject}
}

type tappedUntapRuleEffect struct{ subject PermanentRef }

func (*tappedUntapRuleEffect) Text() string {
	return "prevent untap during untap steps while source remains tapped"
}
func (*tappedUntapRuleEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *tappedUntapRuleEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), battlefieldSource: true, children: []definitionEdge{{"subject", e.subject}, {"source", This}}}
}
func (e *tappedUntapRuleEffect) refersToSourcePermanent() bool { return true }
func (e *tappedUntapRuleEffect) Apply(ctx *EffectContext) error {
	source, ok := This.resolvePermanent(ctx)
	if !ok || !source.Tapped {
		return nil
	}
	subject, ok := e.subject.resolvePermanent(ctx)
	if !ok {
		return nil
	}
	rule := &capturedTappedUntapRule{source: capturedPermanent{source.ID(), source.incarnationID}, subject: capturedPermanent{subject.ID(), subject.incarnationID}}
	rule.SetSourceID(source.ID())
	ctx.Game.effects.Add(rule)
	return nil
}

type capturedTappedUntapRule struct {
	effectSource
	source, subject capturedPermanent
	expired         bool
}

func (*capturedTappedUntapRule) Text() string                         { return "captured untap-step restriction" }
func (*capturedTappedUntapRule) Properties() EffectProperties         { return EffectProperties{} }
func (*capturedTappedUntapRule) GetLayer() core.Layer                 { return core.LayerAbility }
func (*capturedTappedUntapRule) GetDuration() core.Duration           { return core.Indefinite }
func (*capturedTappedUntapRule) ApplyContinuous(*EffectContext) error { return nil }
func (e *capturedTappedUntapRule) IsActive(g *Game) bool {
	source := g.FindPermanent(e.source.id)
	if source == nil || source.incarnationID != e.source.incarnation || !source.Tapped {
		e.expired = true
	}
	return !e.expired
}
func (e *capturedTappedUntapRule) cloneEffect() ContinuousEffect { clone := *e; return &clone }
func (e *capturedTappedUntapRule) subjectAtDecision(g *Game) *Permanent {
	subject := g.FindPermanent(e.subject.id)
	if subject == nil || subject.incarnationID != e.subject.incarnation {
		return nil
	}
	return subject
}
