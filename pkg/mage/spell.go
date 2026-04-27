package mage

import (
	. "git.sr.ht/~cdcarter/mage-go/pkg/mage/core"
	"github.com/google/uuid"
)

// SpellAbility represents a spell's on-resolution effect.
type SpellAbility struct {
	BaseAbility
	effects []Effect
	targets []Target
}

// NewSpellAbility creates a spell ability with the given effects and no targets.
func NewSpellAbility(effects ...Effect) *SpellAbility {
	return &SpellAbility{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilitySpell,
		},
		effects: effects,
	}
}

// NewTargetedSpell creates a spell ability that requires a target.
func NewTargetedSpell(target Target, effects ...Effect) *SpellAbility {
	return &SpellAbility{
		BaseAbility: BaseAbility{
			id:          uuid.New(),
			abilityType: AbilitySpell,
		},
		effects: effects,
		targets: []Target{target},
	}
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

func (sa *SpellAbility) Effects() []Effect { return sa.effects }
func (sa *SpellAbility) Targets() []Target { return sa.targets }
