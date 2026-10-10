package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithLandTypeTextRevision permits checked replacement of a printed land rule.
// The declaration must set one basic land type with one global or attached effect.
func (a *StaticAbilityHolder) WithLandTypeTextRevision() *StaticAbilityHolder {
	a.landTypeTextRevision = true
	return a
}

func (a *StaticAbilityHolder) supportsLandTypeTextRevision() bool {
	return a.landTypeTextRevisionContext() != 0
}

type landTypeRevisionContext uint8

const (
	invalidLandTypeRevision landTypeRevisionContext = iota
	globalLandTypeRevision
	attachedLandTypeRevision
	chosenAttachedLandTypeRevision
)

func (a *StaticAbilityHolder) landTypeTextRevisionContext() landTypeRevisionContext {
	if len(a.Effects) != 1 {
		return 0
	}
	var subtypes []string
	var context landTypeRevisionContext
	switch effect := a.Effects[0].(type) {
	case *basicLandSetEffect:
		if effect == nil {
			return 0
		}
		subtypes, context = effect.subtypes, globalLandTypeRevision
	case *basicLandAttachedEffect:
		if effect == nil || effect.GetLayer() != core.LayerType || effect.GetDuration() != core.WhileOnBattlefield {
			return 0
		}
		if effect.chosen {
			if len(effect.subtypes) != 0 {
				return 0
			}
			return chosenAttachedLandTypeRevision
		}
		subtypes, context = effect.subtypes, attachedLandTypeRevision
	default:
		return 0
	}
	if len(subtypes) != 1 {
		return 0
	}
	if _, valid := basicLandTypeColor(subtypes[0]); !valid {
		return 0
	}
	return context
}

// RevisePrintedStaticLandType replaces one present printed basic-land-type rule.
// The caller must supply text-layer order and lifetime separately.
func (g *Game) RevisePrintedStaticLandType(source *Permanent, abilityID uuid.UUID, changes ...TextChange) error {
	return g.revisePrintedStaticAbility(source, abilityID, func(a *StaticAbilityHolder) bool {
		return a.landTypeTextRevision && a.supportsLandTypeTextRevision()
	}, changes...)
}

// WithLayeredLandTypeTextRevision permits layer 3 changes to a printed land rule.
// Land types and intrinsic mana use the changed rule in layer 4.
func (a *StaticAbilityHolder) WithLayeredLandTypeTextRevision() *StaticAbilityHolder {
	a.WithLandTypeTextRevision()
	a.landTypeTextLayer = true
	return a
}

// AddPrintedStaticLandTypeTextEffect changes one printed land rule in layer 3.
// Each pass restores the saved rule before it applies ordered changes.
func (g *Game) AddPrintedStaticLandTypeTextEffect(source *Permanent, abilityID uuid.UUID, duration core.Duration, changes ...TextChange) error {
	return g.addPrintedStaticTextEffect(source, abilityID, duration, staticLandTypeTextContract, changes...)
}
