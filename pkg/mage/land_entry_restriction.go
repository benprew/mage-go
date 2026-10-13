package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type landEntryRestrictionRule struct{}

// PreventLandEntry stops land entry while the source ability is active.
// Land play permission requires a separate rule.
func PreventLandEntry() StaticRule { return landEntryRestrictionRule{} }

func (landEntryRestrictionRule) staticRule() {}
func (landEntryRestrictionRule) inspectDefinition() definitionNode {
	return definitionNode{Role: "static-rule", Text: "lands cannot enter the battlefield", battlefieldSource: true}
}
func (landEntryRestrictionRule) staticEffects() []ContinuousEffect {
	return []ContinuousEffect{&landEntryRestrictionEffect{FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil }).(*funcContinuousEffect)}}
}

type landEntryRestrictionEffect struct{ *funcContinuousEffect }

func (*landEntryRestrictionEffect) inspectDefinition() definitionNode {
	return landEntryRestrictionRule{}.inspectDefinition()
}
func (e *landEntryRestrictionEffect) cloneEffect() ContinuousEffect {
	clone := *e
	inner := *e.funcContinuousEffect
	clone.funcContinuousEffect = &inner
	return &clone
}

// IsLandEntryProhibited checks active entry rules without changing game state.
func (g *Game) IsLandEntryProhibited(card Card) bool {
	return card != nil && g.isLandEntryProhibited(card.HasType(core.TypeLand))
}

func (g *Game) isLandEntryProhibited(land bool) bool {
	if !land {
		return false
	}
	if g.effects.Rules.LandsCantEnter() {
		return true
	}
	for _, effect := range g.effects.effects {
		if !effect.IsActive(g) {
			continue
		}
		inner := effect
		if printed, ok := inner.(*printedAbilityContinuousEffect); ok {
			inner = printed.ContinuousEffect
		}
		if _, ok := inner.(*landEntryRestrictionEffect); ok {
			source := g.FindPermanent(effect.SourceID())
			if source != nil && !source.FaceDown {
				return true
			}
		}
	}
	return false
}
