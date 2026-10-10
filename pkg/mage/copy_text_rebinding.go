package mage

import "slices"

// Copy exceptions can hide a defective text builder unless its output is checked first.
func rebindCopyTextDefinition(card *BaseCard, modifications []CopyModification) {
	if card.textDefinition == nil || card.textDefinition.build == nil {
		return
	}
	build := card.textDefinition.build
	origin := card.definitionPlan
	saved := slices.Clone(modifications)
	card.textDefinition = &textDefinition{
		changes: slices.Clone(card.textDefinition.changes),
		build: func(changes []TextChange) (func() (Card, error), error) {
			factory, err := build(changes)
			if err != nil {
				return nil, err
			}
			plan, err := newCardDefinitionPlan(factory)
			if err != nil {
				return nil, err
			}
			printed, err := origin.Build()
			if err != nil {
				return nil, err
			}
			return func() (Card, error) {
				changed, err := plan.Build()
				if err != nil {
					return nil, err
				}
				base := changed.(*BaseCard)
				if err := checkTextProtectedCharacteristics(base, printed.(*BaseCard)); err != nil {
					return nil, err
				}
				for _, modification := range saved {
					modification.applyCopyModification(base)
				}
				return base, nil
			}, nil
		},
	}
}
