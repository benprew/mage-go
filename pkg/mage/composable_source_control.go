package mage

import . "github.com/benprew/mage-go/pkg/mage/core"

// GainControlWhileSourceControlled keeps control while you control the original source.
// requireTapped also ends the effect when that source untaps.
func GainControlWhileSourceControlled(subject PermanentRef, requireTapped bool) ResolvingEffect {
	return &sourceControlledEffect{subject: subject, requireTapped: requireTapped}
}

type sourceControlledEffect struct {
	subject       PermanentRef
	requireTapped bool
}

func (e *sourceControlledEffect) Text() string { return "gain control while you control the source" }
func (*sourceControlledEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *sourceControlledEffect) Apply(ctx *EffectContext) error {
	source, sourceOK := This.resolvePermanent(ctx)
	target, targetOK := e.subject.resolvePermanent(ctx)
	if !sourceOK || !targetOK || source.ControllerID() != ctx.Controller || (e.requireTapped && !source.Tapped) {
		return nil
	}
	conditions := []ControlConditionData{ControlSourceControlledByEffectController{}}
	if e.requireTapped {
		conditions = append(conditions, ControlSourceTapped{})
	}
	effect := newControlContinuousEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: target.ID(), ControllerID: ctx.Controller, Duration: Indefinite, Conditions: conditions})
	effect.sourceIncarnation = source.incarnationID
	effect.targetIncarnation = target.incarnationID
	effect.sourceTapped = e.requireTapped
	ctx.Game.effects.Add(effect)
	ctx.Game.ApplyContinuousEffects()
	return nil
}
func (e *sourceControlledEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), battlefieldSource: true, children: []definitionEdge{{"subject", e.subject}}}
}
