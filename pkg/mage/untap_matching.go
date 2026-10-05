package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type matchingUntapRestriction struct {
	ContinuousEffect
	filter PermanentFilter
}

// PreventUntapDuringStepForMatching checks final characteristics at the untap decision.
// Other untap effects can still untap the selected permanents.
func PreventUntapDuringStepForMatching(filter PermanentFilter) ContinuousEffect {
	return &matchingUntapRestriction{
		ContinuousEffect: FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil }),
		filter:           filter,
	}
}

func (g *Game) matchingUntapRestrictions() map[uuid.UUID]bool {
	blocked := make(map[uuid.UUID]bool)
	for _, effect := range g.effects.effects {
		if !effect.IsActive(g) {
			continue
		}
		inner := effect
		if printed, ok := inner.(*printedAbilityContinuousEffect); ok {
			inner = printed.ContinuousEffect
		}
		rule, ok := inner.(*matchingUntapRestriction)
		if !ok {
			continue
		}
		for _, permanent := range g.FilterBattlefield(rule.filter) {
			blocked[permanent.ID()] = true
		}
	}
	return blocked
}
