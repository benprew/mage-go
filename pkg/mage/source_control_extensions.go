package mage

import "github.com/benprew/mage-go/pkg/mage/core"

// GainControlWhileSourceTappedAndPowerLE keeps a live source power limit.
// Source control changes do not end this duration.
func GainControlWhileSourceTappedAndPowerLE(subject PermanentRef) ResolvingEffect {
	return &conditionalSourceControlEffect{subject: subject}
}

// GainControlEachWhileSourceControlled saves all present members at resolution.
// Later membership changes cannot add subjects to this duration.
func GainControlEachWhileSourceControlled(subjects PermanentSet) ResolvingEffect {
	return &conditionalSourceControlEffect{subjects: subjects, controlled: true}
}

type conditionalSourceControlEffect struct {
	subject    PermanentRef
	subjects   PermanentSet
	controlled bool
}

func (*conditionalSourceControlEffect) Text() string {
	return "gain control while source conditions hold"
}
func (*conditionalSourceControlEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeDetriment}
}
func (e *conditionalSourceControlEffect) refersToSourcePermanent() bool { return true }
func (e *conditionalSourceControlEffect) inspectDefinition() definitionNode {
	edge := definitionEdge{"subject", e.subject}
	if e.controlled {
		edge = definitionEdge{"subjects", e.subjects}
	}
	return definitionNode{Role: "resolving-effect", Text: e.Text(), battlefieldSource: true, children: []definitionEdge{edge}}
}
func (e *conditionalSourceControlEffect) Apply(ctx *EffectContext) error {
	source, ok := This.resolvePermanent(ctx)
	if !ok {
		return nil
	}
	conditions := []ControlConditionData{ControlSourceTapped{}, ControlTargetPowerLESource{}}
	var subjects []*Permanent
	if e.controlled {
		if source.ControllerID() != ctx.Controller {
			return nil
		}
		conditions = []ControlConditionData{ControlSourceControlledByEffectController{}}
		subjects = e.subjects.resolvePermanents(ctx)
	} else {
		subject, present := e.subject.resolvePermanent(ctx)
		if !present || !source.Tapped || subject.CurrentPower(ctx.Game) > source.CurrentPower(ctx.Game) {
			return nil
		}
		subjects = []*Permanent{subject}
	}
	seen := make(map[capturedPermanent]bool)
	for _, subject := range subjects {
		if subject == nil || subject.PhasedOut {
			continue
		}
		visit := capturedPermanent{subject.ID(), subject.incarnationID}
		if seen[visit] {
			continue
		}
		seen[visit] = true
		effect := newControlContinuousEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: subject.ID(), ControllerID: ctx.Controller, Duration: core.Indefinite, Conditions: conditions})
		effect.sourceIncarnation = source.incarnationID
		effect.targetIncarnation = subject.incarnationID
		effect.sourceTapped = !e.controlled
		ctx.Game.effects.Add(effect)
	}
	ctx.Game.ApplyContinuousEffects()
	return nil
}
