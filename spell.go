package mage

import "github.com/google/uuid"

// SpellAbility represents a spell's on-resolution effect.
type SpellAbility struct {
	BaseAbility
	Effs []Effect
	Tgts []Target
}

func NewSpellAbility(effects ...Effect) *SpellAbility {
	return &SpellAbility{
		BaseAbility: BaseAbility{
			ID_:   uuid.New(),
			Type_: AbilitySpell,
		},
		Effs: effects,
	}
}

func (sa *SpellAbility) AddTarget(t Target) *SpellAbility {
	sa.Tgts = append(sa.Tgts, t)
	return sa
}

func (sa *SpellAbility) Effects() []Effect { return sa.Effs }
func (sa *SpellAbility) Targets() []Target { return sa.Tgts }
