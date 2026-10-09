package mage

import "github.com/google/uuid"

// WithLandTypeTextRevision permits checked replacement of a printed land rule.
// The declaration must set one basic land type with one global effect.
func (a *StaticAbilityHolder) WithLandTypeTextRevision() *StaticAbilityHolder {
	a.landTypeTextRevision = true
	return a
}

func (a *StaticAbilityHolder) supportsLandTypeTextRevision() bool {
	if len(a.Effects) != 1 {
		return false
	}
	effect, ok := a.Effects[0].(*basicLandSetEffect)
	if !ok || effect == nil || len(effect.subtypes) != 1 {
		return false
	}
	_, valid := basicLandTypeColor(effect.subtypes[0])
	return valid
}

// RevisePrintedStaticLandType replaces one present printed basic-land-type rule.
// The caller must supply text-layer order and lifetime separately.
func (g *Game) RevisePrintedStaticLandType(source *Permanent, abilityID uuid.UUID, changes ...TextChange) error {
	return g.revisePrintedStaticAbility(source, abilityID, func(a *StaticAbilityHolder) bool {
		return a.landTypeTextRevision && a.supportsLandTypeTextRevision()
	}, changes...)
}
