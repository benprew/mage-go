package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// WithAnimationTextRevision permits checked replacement of a printed animation.
// The declaration must contain all layers from one AnimateEach rule.
func (a *StaticAbilityHolder) WithAnimationTextRevision() *StaticAbilityHolder {
	a.animationTextRevision = true
	return a
}

func (a *StaticAbilityHolder) supportsAnimationTextRevision() bool {
	if len(a.Effects) < 2 {
		return false
	}
	first, ok := a.Effects[0].(*staticAnimationLayer)
	if !ok || first == nil || first.rule == nil {
		return false
	}
	layers := []core.Layer{core.LayerType, core.LayerPT}
	if first.rule.setColors {
		layers = []core.Layer{core.LayerType, core.LayerColor, core.LayerPT}
	}
	if len(a.Effects) != len(layers) {
		return false
	}
	for i, effect := range a.Effects {
		part, ok := effect.(*staticAnimationLayer)
		if !ok || part == nil || part.rule != first.rule || part.layer != layers[i] {
			return false
		}
	}
	return true
}

// RevisePrintedStaticAnimation replaces one present printed animation.
// The caller must supply text-layer order and lifetime separately.
func (g *Game) RevisePrintedStaticAnimation(source *Permanent, abilityID uuid.UUID, changes ...TextChange) error {
	return g.revisePrintedStaticAbility(source, abilityID, func(a *StaticAbilityHolder) bool {
		return a.animationTextRevision && a.supportsAnimationTextRevision()
	}, changes...)
}
