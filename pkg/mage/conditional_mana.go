package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type conditionalManaPlan struct {
	condition           StateCondition
	whenTrue, whenFalse []ManaProduction
}

// NewConditionalManaAbility uses the same condition for payment queries and mana production.
// Both branches require fixed colors and positive amounts.
// The condition must stay the same when the source taps because payment queries precede costs.
func NewConditionalManaAbility(condition StateCondition, whenTrue, whenFalse []ManaProduction) *ManaAbility {
	ability := NewManaAbility(core.Colorless)
	ability.Productions = nil
	ability.conditional = &conditionalManaPlan{condition: condition, whenTrue: slices.Clone(whenTrue), whenFalse: slices.Clone(whenFalse)}
	return ability
}

func (p *conditionalManaPlan) productions(g GameReader, sourceID uuid.UUID) []ManaProduction {
	if g == nil || nilDefinition(p.condition) {
		return nil
	}
	source := g.FindPermanent(sourceID)
	if source == nil || source.PhasedOut {
		return nil
	}
	selected := p.whenFalse
	if p.condition.checkState(g, StateContext{SourceID: sourceID, ControllerID: source.ControllerID(), SourceIncarnation: source.incarnationID}) {
		selected = p.whenTrue
	}
	return slices.Clone(selected)
}

func (p *conditionalManaPlan) inspectDefinition() definitionNode {
	node := definitionNode{Role: "conditional-mana", children: []definitionEdge{{"condition", p.condition}}}
	for _, branch := range [][]ManaProduction{p.whenTrue, p.whenFalse} {
		if len(branch) == 0 {
			node.problem = "supply mana in both branches"
		}
		for _, production := range branch {
			if production.Color < core.Colorless || production.Color >= core.AnyColor || production.Amount <= 0 || production.AnyCombination || production.Restriction != nil {
				node.problem = "supply fixed colors and positive amounts without restrictions"
			}
		}
	}
	return node
}
