package mage

import (
	"fmt"
	"slices"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// CopyModification changes characteristics within the copy process.
// The resulting characteristics are available to subsequent copies.
type CopyModification interface {
	validateCopyModification() error
	applyCopyModification(*BaseCard)
}

// ColorCharacteristicDefinition identifies an effect that defines color.
// Ordinary color changes must not implement this contract.
type ColorCharacteristicDefinition interface {
	DefinesColor() bool
}

type copyRetainedColors struct{ colors []core.Color }

// CopyRetainColors keeps saved colors as part of the copied characteristics.
// An empty list keeps the copy colorless. Color definitions are not copied.
func CopyRetainColors(colors ...core.Color) CopyModification {
	saved := make([]core.Color, 0, len(colors))
	for _, color := range colors {
		if color != core.Colorless && !slices.Contains(saved, color) {
			saved = append(saved, color)
		}
	}
	return copyRetainedColors{colors: saved}
}

func (m copyRetainedColors) validateCopyModification() error {
	for _, color := range m.colors {
		switch color {
		case core.White, core.Blue, core.Black, core.Red, core.Green:
		default:
			return fmt.Errorf("retained copy colors require Magic colors; use an empty list for colorless")
		}
	}
	return nil
}

func (m copyRetainedColors) applyCopyModification(card *BaseCard) {
	card.colorOverride = slices.Clone(m.colors)
	card.abilities = slices.DeleteFunc(card.abilities, func(ability Ability) bool {
		holder, ok := UnwrapAbility(ability).(*StaticAbilityHolder)
		if !ok {
			return false
		}
		for _, effect := range holder.Effects {
			if definition, ok := effect.(ColorCharacteristicDefinition); ok && definition.DefinesColor() {
				delete(card.abilityKeys, ability.AbilityID())
				return true
			}
		}
		return false
	})
}

type copyTypeAddition struct{ types []core.CardType }

// CopyAddCardTypes adds permanent types to the copied characteristics.
// It keeps existing types and abilities, including characteristic definitions.
func CopyAddCardTypes(types ...core.CardType) CopyModification {
	return copyTypeAddition{types: slices.Clone(types)}
}

func (m copyTypeAddition) validateCopyModification() error {
	if len(m.types) == 0 {
		return fmt.Errorf("copy type addition requires at least one permanent type")
	}
	for _, kind := range m.types {
		switch kind {
		case core.TypeCreature, core.TypeArtifact, core.TypeEnchantment, core.TypeLand, core.TypePlaneswalker:
		default:
			return fmt.Errorf("copy type addition requires permanent types")
		}
	}
	return nil
}

func (m copyTypeAddition) applyCopyModification(card *BaseCard) {
	for _, kind := range m.types {
		if !slices.Contains(card.types, kind) {
			card.types = append(card.types, kind)
		}
	}
}

func validateCopyModifications(modifications []CopyModification) error {
	for _, modification := range modifications {
		if modification == nil {
			return fmt.Errorf("copy modification must not be nil")
		}
		if err := modification.validateCopyModification(); err != nil {
			return err
		}
	}
	return nil
}

// WithModifications creates a validated snapshot with copy exceptions.
// Each build has fresh rules. The original snapshot does not change.
func (v *CopiableValues) WithModifications(modifications ...CopyModification) (*CopiableValues, error) {
	if v == nil {
		return nil, fmt.Errorf("copy modifications require a snapshot")
	}
	if err := validateCopyModifications(modifications); err != nil {
		return nil, err
	}
	if len(modifications) == 0 {
		return v, nil
	}
	saved := slices.Clone(modifications)
	plan, err := newCardDefinitionPlan(func() (Card, error) {
		card, err := v.Build()
		if err != nil {
			return nil, err
		}
		base := card.(*BaseCard)
		rebindCopyTextDefinition(base, saved)
		for _, modification := range saved {
			modification.applyCopyModification(base)
		}
		return card, nil
	})
	if err != nil {
		return nil, err
	}
	// Copy exceptions must not give existing entry rules another use in one event.
	plan.declarationOrigin = v.definition
	if plan.declarationOrigin != nil && plan.declarationOrigin.declarationOrigin != nil {
		plan.declarationOrigin = plan.declarationOrigin.declarationOrigin
	}
	return plan.CopiableValues(), nil
}
