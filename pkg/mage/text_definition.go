package mage

import (
	"fmt"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// TextChange retains one validated color or basic land word substitution.
type TextChange struct {
	kind     uint8
	from, to int
}

// ReplaceColorWord requires two different Magic color words.
func ReplaceColorWord(from, to core.Color) (TextChange, error) {
	if from < core.White || from > core.Green || to < core.White || to > core.Green || from == to {
		return TextChange{}, fmt.Errorf("text change requires two different color words")
	}
	return TextChange{1, int(from), int(to)}, nil
}

// ReplaceBasicLandWord requires two different basic land type words.
func ReplaceBasicLandWord(from, to core.BasicLandType) (TextChange, error) {
	if from.String() == "" || to.String() == "" || from == to {
		return TextChange{}, fmt.Errorf("text change requires two different basic land words")
	}
	return TextChange{2, int(from), int(to)}, nil
}

// ApplyColor changes a word that the caller has identified as a color word.
func (c TextChange) ApplyColor(word core.Color) core.Color {
	if c.kind == 1 && int(word) == c.from {
		return core.Color(c.to)
	}
	return word
}

// ApplyBasicLand changes a word that the caller has identified as a land type.
func (c TextChange) ApplyBasicLand(word core.BasicLandType) core.BasicLandType {
	if c.kind == 2 && int(word) == c.from {
		return core.BasicLandType(c.to)
	}
	return word
}

type textDefinition struct {
	build   func([]TextChange) (func() (Card, error), error)
	changes []TextChange
}

// WithTextDefinitionBuilder retains a pure builder for independent text prototypes.
// The builder must keep names, mana symbols and color indicators unchanged.
func WithTextDefinitionBuilder(build func([]TextChange) (CardFactory, error)) CardOption {
	return func(card *BaseCard) {
		card.textDefinition = &textDefinition{}
		if build == nil {
			return
		}
		card.textDefinition.build = func(changes []TextChange) (func() (Card, error), error) {
			factory, err := build(changes)
			if err != nil || factory == nil {
				return nil, err
			}
			return func() (Card, error) { return factory(), nil }, nil
		}
	}
}

// BuildTextChangedDefinition builds new rules without changing a game object.
// It keeps printed copy information separate from the changed text.
func BuildTextChangedDefinition(card Card, changes ...TextChange) (Card, error) {
	base, ok := card.(*BaseCard)
	if !ok || base == nil || base.textDefinition == nil || base.textDefinition.build == nil || base.definitionPlan == nil {
		return nil, fmt.Errorf("card definition has no text builder")
	}
	ordered := slices.Clone(base.textDefinition.changes)
	for _, change := range changes {
		if change.kind != 1 && change.kind != 2 {
			return nil, fmt.Errorf("text change requires validated words")
		}
		ordered = append(ordered, change)
	}
	factory, err := base.textDefinition.build(slices.Clone(ordered))
	if err != nil {
		return nil, err
	}
	plan, err := newCardDefinitionPlan(factory)
	if err != nil {
		return nil, err
	}
	built, err := plan.Build()
	if err != nil {
		return nil, err
	}
	result := built.(*BaseCard)
	printed, err := base.definitionPlan.Build()
	if err != nil {
		return nil, err
	}
	if err := checkTextProtectedCharacteristics(result, printed.(*BaseCard)); err != nil {
		return nil, err
	}
	result.definitionPlan = base.definitionPlan
	result.printedDefinitionPlan = base.printedDefinitionPlan
	result.copiableValues = base.copiableValues
	result.textDefinition = &textDefinition{build: base.textDefinition.build, changes: ordered}
	return result, nil
}

func checkTextProtectedCharacteristics(result, printed *BaseCard) error {
	if result.Name() != printed.Name() || result.ManaCost().String() != printed.ManaCost().String() || result.HasManaCost() != printed.HasManaCost() || !slices.Equal(result.colorOverride, printed.colorOverride) || (result.colorOverride == nil) != (printed.colorOverride == nil) || !slices.Equal(result.Types(), printed.Types()) || !slices.Equal(result.SuperTypes(), printed.SuperTypes()) || result.Power() != printed.Power() || result.Toughness() != printed.Toughness() {
		return fmt.Errorf("text builder changed protected printed characteristics")
	}
	return nil
}

// TextChangeRef reads a typed substitution from the current resolution frame.
type TextChangeRef interface {
	Resolve(*EffectContext) (TextChange, bool)
	inspectDefinition() definitionNode
}

type colorWordSubstitution struct{ from, to ColorWordRef }
type landWordSubstitution struct{ from, to BasicLandWordRef }

// ColorWordSubstitution reads two color words without making choices.
func ColorWordSubstitution(from, to ColorWordRef) TextChangeRef {
	return colorWordSubstitution{from, to}
}

// BasicLandWordSubstitution reads two land words without making choices.
func BasicLandWordSubstitution(from, to BasicLandWordRef) TextChangeRef {
	return landWordSubstitution{from, to}
}

func (r colorWordSubstitution) Resolve(ctx *EffectContext) (TextChange, bool) {
	if nilDefinition(r.from) || nilDefinition(r.to) {
		return TextChange{}, false
	}
	from, first := r.from.Resolve(ctx)
	to, second := r.to.Resolve(ctx)
	if !first || !second {
		return TextChange{}, false
	}
	change, err := ReplaceColorWord(from, to)
	return change, err == nil
}
func (r landWordSubstitution) Resolve(ctx *EffectContext) (TextChange, bool) {
	if nilDefinition(r.from) || nilDefinition(r.to) {
		return TextChange{}, false
	}
	from, first := r.from.Resolve(ctx)
	to, second := r.to.Resolve(ctx)
	if !first || !second {
		return TextChange{}, false
	}
	change, err := ReplaceBasicLandWord(from, to)
	return change, err == nil
}
func (r colorWordSubstitution) inspectDefinition() definitionNode {
	return definitionNode{Role: "color-word-substitution", children: []definitionEdge{{"from", r.from}, {"to", r.to}}}
}
func (r landWordSubstitution) inspectDefinition() definitionNode {
	return definitionNode{Role: "basic-land-word-substitution", children: []definitionEdge{{"from", r.from}, {"to", r.to}}}
}
