package mage

import "github.com/benprew/mage-go/pkg/mage/core"

type hostPTEffect struct {
	*composedLayerEffect
	landType     string
	creatureType string
}

// CreaturesPTWhileAttachedToBasicLand modifies matching creatures' P/T while
// the current host is Basic and has the supplied basic land type.
func CreaturesPTWhileAttachedToBasicLand(landType, creatureType string, power, toughness int) ContinuousEffect {
	basic := NewPermanentFilter("basic", func(p *Permanent, _ *Game) bool {
		return p.Card.HasSuperType(core.SuperBasic)
	})
	condition := PermanentMatchesFilter(AttachedPermanent, And(basic, HasSubType(landType)))
	subjects := MatchingPermanents(And(IsCreature, HasSubType(creatureType)))
	rule := While(condition, ModifyEachPT(subjects, Fixed(power), Fixed(toughness)))
	return &hostPTEffect{
		composedLayerEffect: rule.staticEffects()[0].(*composedLayerEffect),
		landType:            landType,
		creatureType:        creatureType,
	}
}

func (e *hostPTEffect) cloneEffect() ContinuousEffect {
	clone := *e
	clone.composedLayerEffect = e.composedLayerEffect.cloneEffect().(*composedLayerEffect)
	return &clone
}

func (e *hostPTEffect) validContext() bool {
	if e == nil || e.composedLayerEffect == nil {
		return false
	}
	_, valid := basicLandTypeColor(e.landType)
	return valid && ValidateSubtypeFamily(SubtypeCreature, e.creatureType) == nil
}

func (e *hostPTEffect) inspectDefinition() definitionNode {
	n := e.composedLayerEffect.inspectDefinition()
	if !e.validContext() {
		n.problem = "host P/T requires one basic land type and a nonempty creature subtype"
	}
	return n
}
