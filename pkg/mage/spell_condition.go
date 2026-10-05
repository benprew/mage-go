package mage

import (
	"fmt"
	"slices"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

// WithCastingCondition requires a pure condition before casting choices and payment.
func WithCastingCondition(condition StateCondition) ActionOption {
	return func(a *ActionDefinition) {
		a.castingConditions = append(slices.Clone(a.castingConditions), condition)
	}
}

type castingConditionDefinition struct{ condition StateCondition }

func (c castingConditionDefinition) inspectDefinition() definitionNode {
	return definitionNode{Role: "casting-condition", children: []definitionEdge{{"condition", c.condition}}}
}

type stepCondition struct{ steps []PhaseStep }

// IsDuringSteps checks the current step without granting priority or casting permission.
func IsDuringSteps(steps ...PhaseStep) StateCondition {
	return stepCondition{steps: slices.Clone(steps)}
}

func (stepCondition) stateCondition() {}
func (c stepCondition) checkState(g GameReader, _ StateContext) bool {
	return slices.Contains(c.steps, g.GetStep())
}
func (c stepCondition) inspectDefinition() definitionNode {
	n := definitionNode{Role: "state-condition", Text: "during the specified steps"}
	if len(c.steps) == 0 {
		n.problem = "supply at least one step"
	}
	for _, step := range c.steps {
		if step < Untap || step > Cleanup {
			n.problem = "supply valid turn steps"
		}
	}
	return n
}

func (g *Game) checkSpellCastingConditions(card Card, caster uuid.UUID) error {
	for _, ability := range card.Abilities() {
		action, ok := UnwrapAbility(ability).(*ActionDefinition)
		if !ok || action.kind != ActionSpell {
			continue
		}
		for _, condition := range action.castingConditions {
			if nilDefinition(condition) || !condition.checkState(g, StateContext{SourceID: card.ID(), ControllerID: caster}) {
				return fmt.Errorf("casting condition for %s is not satisfied", card.Name())
			}
		}
	}
	return nil
}

type afterFirstUpkeepCondition struct{}

// AfterFirstUpkeep permits later upkeeps after the first upkeep ends.
func AfterFirstUpkeep() StateCondition            { return afterFirstUpkeepCondition{} }
func (afterFirstUpkeepCondition) stateCondition() {}
func (afterFirstUpkeepCondition) checkState(g GameReader, _ StateContext) bool {
	return g.UpkeepHasEnded()
}
func (afterFirstUpkeepCondition) inspectDefinition() definitionNode {
	return definitionNode{Role: "state-condition", Text: "after the first upkeep"}
}
