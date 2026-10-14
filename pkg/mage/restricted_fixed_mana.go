package mage

import (
	"slices"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type restrictedFixedManaEffect struct {
	productions []ManaProduction
	restriction ManaRestriction
}

// AddRestrictedMana keeps fixed productions in one event with one spending rule.
func AddRestrictedMana(productions []ManaProduction, restriction ManaRestriction) ResolvingEffect {
	saved := slices.Clone(productions)
	for i := range saved {
		saved[i].Restriction = restriction
	}
	return &restrictedFixedManaEffect{productions: saved, restriction: restriction}
}

func (*restrictedFixedManaEffect) Text() string                 { return "add restricted fixed mana" }
func (*restrictedFixedManaEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *restrictedFixedManaEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text()}
	if len(e.productions) == 0 || nilDefinition(e.restriction) {
		n.problem = "supply fixed mana productions and a spending restriction"
	}
	for _, production := range e.productions {
		if production.Color < Colorless || production.Color >= AnyColor || production.Amount <= 0 || production.AnyCombination {
			n.problem = "supply concrete mana colors and positive fixed amounts"
		}
	}
	return n
}
func (e *restrictedFixedManaEffect) Apply(ctx *EffectContext) error {
	return addEffectMana(ctx, slices.Clone(e.productions))
}
