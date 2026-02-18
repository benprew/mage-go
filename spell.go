package mage

import "github.com/google/uuid"

// SpellAbility represents a spell's on-resolution effect.
type SpellAbility struct {
	BaseAbility
	effects []Effect
	targets []Target
}

func NewSpellAbility(effects ...Effect) *SpellAbility {
	return &SpellAbility{
		BaseAbility: BaseAbility{
			id:   uuid.New(),
			abilityType: AbilitySpell,
		},
		effects: effects,
	}
}

func (sa *SpellAbility) AddTarget(t Target) *SpellAbility {
	sa.targets = append(sa.targets, t)
	return sa
}

func (sa *SpellAbility) Effects() []Effect { return sa.effects }
func (sa *SpellAbility) Targets() []Target { return sa.targets }
