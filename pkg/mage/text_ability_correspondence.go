package mage

import (
	"fmt"
	"maps"

	"github.com/google/uuid"
)

type definitionAbilityKey struct {
	group, member int
}

// WithDefinitionAbilityGroup identifies the abilities made by one declaration.
// Group numbers must stay the same in each build of the definition.
func WithDefinitionAbilityGroup(group int, option CardOption) CardOption {
	return func(card *BaseCard) {
		if card.abilityGroups == nil {
			card.abilityGroups = make(map[int]bool)
		}
		if group < 0 || option == nil || card.abilityGroups[group] {
			card.abilityGroupProblem = "definition ability groups require distinct nonnegative numbers and non-nil options"
			return
		}
		card.abilityGroups[group] = true
		start := len(card.abilities)
		option(card)
		if len(card.abilities) < start {
			card.abilityGroupProblem = "definition ability group removed an earlier declaration"
			return
		}
		if card.abilityKeys == nil {
			card.abilityKeys = make(map[uuid.UUID]definitionAbilityKey)
		}
		for i, ability := range card.abilities[start:] {
			if nilDefinition(ability) {
				card.abilityGroupProblem = "definition ability group contains a nil ability"
				continue
			}
			if _, exists := card.abilityKeys[ability.AbilityID()]; exists {
				card.abilityGroupProblem = "definition ability belongs to more than one group"
			}
			card.abilityKeys[ability.AbilityID()] = definitionAbilityKey{group, i}
		}
	}
}

// TextDefinitionRevision pairs fresh rules with the original ability identities.
// It does not change a game object or transfer saved choices.
type TextDefinitionRevision struct {
	definition   Card
	replacements map[uuid.UUID]Ability
}

// Definition returns the independent changed definition.
func (r *TextDefinitionRevision) Definition() Card { return r.definition }

// ReplacementAbility reads the new declaration for one original ability.
func (r *TextDefinitionRevision) ReplacementAbility(id uuid.UUID) (Ability, bool) {
	if r == nil {
		return nil, false
	}
	ability, ok := r.replacements[id]
	return ability, ok
}

// BuildTextDefinitionRevision checks declaration correspondence before use.
// A changed group cannot silently replace an unrelated ability.
func BuildTextDefinitionRevision(card Card, changes ...TextChange) (*TextDefinitionRevision, error) {
	changed, err := BuildTextChangedDefinition(card, changes...)
	if err != nil {
		return nil, err
	}
	original := card.(*BaseCard)
	rebuilt := changed.(*BaseCard)
	first, err := definitionAbilitiesByKey(original, true)
	if err != nil {
		return nil, err
	}
	second, err := definitionAbilitiesByKey(rebuilt, true)
	if err != nil {
		return nil, err
	}
	if len(first) != len(second) || !maps.Equal(original.abilityGroups, rebuilt.abilityGroups) {
		return nil, fmt.Errorf("text revision changed definition ability groups")
	}
	revision := &TextDefinitionRevision{definition: changed, replacements: make(map[uuid.UUID]Ability, len(first))}
	for key, ability := range first {
		replacement, ok := second[key]
		if !ok || ability.Type() != replacement.Type() {
			return nil, fmt.Errorf("text revision changed definition ability group %d member %d", key.group, key.member)
		}
		revision.replacements[ability.AbilityID()] = replacement
	}
	return revision, nil
}

func definitionAbilitiesByKey(card *BaseCard, requireAll bool) (map[definitionAbilityKey]Ability, error) {
	if card.abilityGroupProblem != "" {
		return nil, fmt.Errorf("%s", card.abilityGroupProblem)
	}
	result := make(map[definitionAbilityKey]Ability, len(card.abilities))
	for _, ability := range card.abilities {
		if nilDefinition(ability) {
			return nil, fmt.Errorf("definition ability correspondence requires non-nil abilities")
		}
		key, ok := card.abilityKeys[ability.AbilityID()]
		if !ok && !requireAll {
			continue
		}
		if !ok || !card.abilityGroups[key.group] || key.group < 0 || key.member < 0 {
			return nil, fmt.Errorf("definition ability has no valid declaration group")
		}
		if result[key] != nil {
			return nil, fmt.Errorf("definition ability group has a duplicate member")
		}
		result[key] = ability
	}
	if len(card.abilityKeys) != len(result) {
		return nil, fmt.Errorf("definition ability groups contain stale declarations")
	}
	return result, nil
}
