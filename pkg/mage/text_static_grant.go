package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithGrantTextRevision permits checked replacement of a printed static grant.
// An ability-layer rule can also modify P/T on the battlefield.
func (a *StaticAbilityHolder) WithGrantTextRevision() *StaticAbilityHolder {
	a.grantTextRevision = true
	return a
}

func (a *StaticAbilityHolder) supportsGrantTextRevision() bool {
	hasAbility := false
	for _, effect := range a.Effects {
		if nilDefinition(effect) || effect.GetDuration() != core.WhileOnBattlefield {
			return false
		}
		switch effect.GetLayer() {
		case core.LayerAbility:
			hasAbility = true
		case core.LayerPT:
			if getEffectSublayer(effect) != core.SublayerPT_Modify {
				return false
			}
		default:
			return false
		}
	}
	return hasAbility
}

// RevisePrintedStaticGrant replaces one present printed static grant.
// The caller must supply text-layer order and lifetime separately.
func (g *Game) RevisePrintedStaticGrant(source *Permanent, abilityID uuid.UUID, changes ...TextChange) error {
	return g.revisePrintedStaticAbility(source, abilityID, func(a *StaticAbilityHolder) bool {
		return a.grantTextRevision && a.supportsGrantTextRevision()
	}, changes...)
}

// WithLayeredGrantTextRevision permits layer 3 changes to a printed global grant.
// The same subjects receive its ability and P/T parts in each layer pass.
func (a *StaticAbilityHolder) WithLayeredGrantTextRevision() *StaticAbilityHolder {
	a.WithGrantTextRevision()
	a.grantTextLayer = true
	return a
}

// AddPrintedStaticGrantTextEffect changes one printed global grant in layer 3.
// Each pass restores the saved rule before it applies ordered changes.
func (g *Game) AddPrintedStaticGrantTextEffect(source *Permanent, abilityID uuid.UUID, duration core.Duration, changes ...TextChange) error {
	return g.addPrintedStaticTextEffect(source, abilityID, duration, true, changes...)
}
