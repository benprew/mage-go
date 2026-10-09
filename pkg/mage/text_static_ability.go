package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithAbilityTextRevision permits checked replacement of a printed ability-layer rule.
// Each effect must use the ability layer while its source is on the battlefield.
func (a *StaticAbilityHolder) WithAbilityTextRevision() *StaticAbilityHolder {
	a.abilityTextRevision = true
	return a
}

func (a *StaticAbilityHolder) supportsAbilityTextRevision() bool {
	if len(a.Effects) == 0 {
		return false
	}
	for _, effect := range a.Effects {
		if nilDefinition(effect) || effect.GetLayer() != core.LayerAbility || effect.GetDuration() != core.WhileOnBattlefield {
			return false
		}
	}
	return true
}

// RevisePrintedStaticAbility replaces one present printed ability-layer rule.
// The caller must supply text-layer order and lifetime separately.
func (g *Game) RevisePrintedStaticAbility(source *Permanent, abilityID uuid.UUID, changes ...TextChange) error {
	return g.revisePrintedStaticAbility(source, abilityID, func(a *StaticAbilityHolder) bool {
		return a.abilityTextRevision && a.supportsAbilityTextRevision()
	}, changes...)
}
