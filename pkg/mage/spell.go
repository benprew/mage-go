package mage

import "github.com/google/uuid"

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

func (sa *SpellAbility) Effects() []Effect { return sa.effects }
func (sa *SpellAbility) Targets() []Target { return sa.targets }
