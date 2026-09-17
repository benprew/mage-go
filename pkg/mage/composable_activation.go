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
	for _, prior := range a.dslRestrictions {
		if prior.kind == o.kind && o.kind != "condition" {
			panic("activation: conflicting " + o.kind + " options")
		}
	}
	switch o.kind {
	case "timing":
		if o.timing == TimingStepOnly && (o.step < Untap || o.step > Cleanup) {
			panic("activation: supply a valid turn step")
		}
		a.timing, a.stepOnly = o.timing, o.step
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

// OnlyAsSorcery requires a main phase, the player's turn, and an empty stack.
func OnlyAsSorcery() ActivationOption {
	return activationRestriction{kind: "timing", timing: TimingSorcery}
}

// OnlyDuring restricts activation to the specified turn step.
func OnlyDuring(step PhaseStep) ActivationOption {
	return activationRestriction{kind: "timing", timing: TimingStepOnly, step: step}
}

// OnlyOnYourTurn restricts activation to the controller's turn.
func OnlyOnYourTurn() ActivationOption {
	return activationRestriction{kind: "timing", timing: YourTurnOnly}
}

// ActivationLimit uses the existing per-turn activation counter.
func ActivationLimit(maximum int) ActivationOption {
	return activationRestriction{kind: "limit", limit: maximum}
}

// ActivationIf checks a pure condition before payment through the activation path.
func ActivationIf(condition StateCondition) ActivationOption {
	return activationRestriction{kind: "condition", condition: condition}
}

func (o activationRestriction) inspectDefinition() definitionNode {
	n := definitionNode{Role: "activation-" + o.kind}
	if o.kind == "condition" {
		n.children = []definitionEdge{{"condition", o.condition}}
	}
	return n
}
