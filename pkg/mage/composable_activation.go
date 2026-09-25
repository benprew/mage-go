package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type activationRestriction struct {
	kind      string
	timing    TimingRule
	step      PhaseStep
	limit     int
	condition StateCondition
}

func (activationRestriction) activationOption() {}
func (o activationRestriction) applyActivation(a *ActionDefinition) {
	combinedTiming := false
	step := o.step
	for _, prior := range a.dslRestrictions {
		if prior.kind == o.kind && o.kind != "condition" {
			if o.kind == "timing" && ((prior.timing == TimingStepOnly && o.timing == YourTurnOnly) || (prior.timing == YourTurnOnly && o.timing == TimingStepOnly)) {
				combinedTiming = true
				if prior.timing == TimingStepOnly {
					step = prior.step
				}
				continue
			}
			panic("activation: conflicting " + o.kind + " options")
		}
	}
	switch o.kind {
	case "timing":
		if o.timing == TimingStepOnly && (o.step < Untap || o.step > Cleanup) {
			panic("activation: supply a valid turn step")
		}
		a.timing, a.stepOnly = o.timing, o.step
		if combinedTiming {
			a.timing, a.stepOnly = TimingStepOnly, step
			a.activationConds = append(a.activationConds, func(g *Game, _ *Permanent, controller uuid.UUID) bool {
				return g.ActivePlayerObj().PlayerID() == controller
			})
		}
	case "limit":
		if o.limit <= 0 {
			panic("activation: supply a positive limit")
		}
		a.limits.MaxActivationsPerTurn = o.limit
	case "condition":
		if nilDefinition(o.condition) {
			panic("activation: supply a condition")
		}
		a.activationConds = append(a.activationConds, func(g *Game, source *Permanent, controller uuid.UUID) bool {
			var id uuid.UUID
			if source != nil {
				id = source.ID()
			}
			return o.condition.checkState(g, StateContext{SourceID: id, ControllerID: controller})
		})
	}
	a.dslRestrictions = append(a.dslRestrictions, o)
}

// ActivationTimingOption sets timing for an activation or a static grant.
type ActivationTimingOption interface {
	ActivationOption
	GrantedActivationOption
	ModalActivationOption
}

type activationTiming struct{ activationRestriction }

func (activationTiming) grantedActivationOption() {}
func (o activationTiming) applyGrantedActivation(a *ActionDefinition) {
	o.applyActivation(a)
}

// OnlyAsSorcery requires a main phase, the player's turn, and an empty stack.
func OnlyAsSorcery() ActivationTimingOption {
	return activationTiming{activationRestriction{kind: "timing", timing: TimingSorcery}}
}

// StepTimingOption restricts a spell or activation to one turn step.
type StepTimingOption interface {
	ActivationTimingOption
	SpellOption
}

type stepTiming struct{ activationTiming }

func (stepTiming) spellOption() {}
func (o stepTiming) applySpell(a *ActionDefinition) {
	o.applyActivation(a)
}

// OnlyDuring restricts casting or activation to the specified turn step.
func OnlyDuring(step PhaseStep) StepTimingOption {
	return stepTiming{activationTiming{activationRestriction{kind: "timing", timing: TimingStepOnly, step: step}}}
}

// OnlyOnYourTurn restricts activation to the controller's turn.
func OnlyOnYourTurn() ActivationTimingOption {
	return activationTiming{activationRestriction{kind: "timing", timing: YourTurnOnly}}
}

// ActivationLimitOption sets a per-turn limit for an activation or a static grant.
type ActivationLimitOption interface {
	ActivationOption
	GrantedActivationOption
	ModalActivationOption
}

type activationLimit struct{ activationRestriction }

func (activationLimit) grantedActivationOption() {}
func (o activationLimit) applyGrantedActivation(a *ActionDefinition) {
	o.applyActivation(a)
}

// ActivationLimit restricts the number of activations each turn.
func ActivationLimit(maximum int) ActivationLimitOption {
	return activationLimit{activationRestriction{kind: "limit", limit: maximum}}
}

// ActivationConditionOption sets a condition for an activation or a static grant.
type ActivationConditionOption interface {
	ActivationOption
	GrantedActivationOption
	ModalActivationOption
}

type activationStateCondition struct{ activationRestriction }

func (activationStateCondition) grantedActivationOption() {}
func (o activationStateCondition) applyGrantedActivation(a *ActionDefinition) {
	o.applyActivation(a)
}

// ActivationIf checks a pure condition before payment through the activation path.
func ActivationIf(condition StateCondition) ActivationConditionOption {
	return activationStateCondition{activationRestriction{kind: "condition", condition: condition}}
}

func (o activationRestriction) inspectDefinition() definitionNode {
	n := definitionNode{Role: "activation-" + o.kind}
	if o.kind == "condition" {
		n.children = []definitionEdge{{"condition", o.condition}}
	}
	return n
}

func (activationTiming) modalActivationOption()                     {}
func (o activationTiming) applyModalActivation(a *ActionDefinition) { o.applyActivation(a) }

func (activationLimit) modalActivationOption()                     {}
func (o activationLimit) applyModalActivation(a *ActionDefinition) { o.applyActivation(a) }

func (activationStateCondition) modalActivationOption()                     {}
func (o activationStateCondition) applyModalActivation(a *ActionDefinition) { o.applyActivation(a) }

type spellStepDefinition struct{ step PhaseStep }

func (s spellStepDefinition) inspectDefinition() definitionNode {
	return definitionNode{Role: "spell-step", Text: s.step.String()}
}
