package mage

import (
	. "github.com/benprew/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

// SpellAbility is kept as a compatibility name for spell actions.
type SpellAbility = ActionDefinition

// NewSpell creates a spell action. Arguments may be Effect values or ActionOption
// values, allowing card code to use one builder for effects and targets.
func NewSpell(parts ...any) *ActionDefinition {
	return NewAction(ActionSpell, actionPartsToOptions(parts...)...)
}

// NewSpellAbility creates a spell ability with the given effects and no targets.
func NewSpellAbility(effects ...Effect) *SpellAbility {
	parts := make([]any, 0, len(effects))
	for _, effect := range effects {
		parts = append(parts, effect)
	}
	return NewSpell(parts...)
}

// NewTargetedSpell creates a spell ability that requires a target.
func NewTargetedSpell(target Target, effects ...Effect) *SpellAbility {
	parts := make([]any, 0, len(effects)+1)
	for _, effect := range effects {
		parts = append(parts, effect)
	}
	parts = append(parts, WithTarget(target))
	return NewSpell(parts...)
}

// NewMultiTargetSpell creates a spell ability that declares multiple Target
// requirements. Each Target may carry its own min/max and predicate, so this
// constructor covers:
//   - "up to N target X" (a single Target with min=0, max=N)
//   - "target X you control and target Y an opponent controls" (two Targets
//     with disjoint predicates)
//   - "any number of target X" (min=0, max=large)
//
// The flat list of chosen UUIDs from all Targets, in declaration order, is
// exposed to effects via ctx.Targets. Effects that act on every chosen target
// should select with ToAllTargets().
func NewMultiTargetSpell(targets []Target, effects ...Effect) *SpellAbility {
	return &SpellAbility{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilitySpell,
		},
		effects: effects,
		targets: targets,
	}
}
