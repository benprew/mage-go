package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type counterPreventionContinuous struct {
	effectSource
	counterType core.CounterType
	ruleID      uuid.UUID
}

func (*counterPreventionContinuous) Text() string { return "prevent damage by removing counters" }
func (*counterPreventionContinuous) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (*counterPreventionContinuous) GetLayer() core.Layer       { return core.LayerAbility }
func (*counterPreventionContinuous) GetDuration() core.Duration { return core.WhileOnBattlefield }
func (e *counterPreventionContinuous) IsActive(g *Game) bool {
	source := g.FindPermanent(e.sourceID)
	return source != nil && !source.PhasedOut
}
func (e *counterPreventionContinuous) ApplyContinuous(ctx *EffectContext) error {
	source := ctx.Game.FindPermanent(e.sourceID)
	if source == nil || source.PhasedOut {
		return nil
	}
	ctx.Game.effects.AddCycleReplacement(&counterDamagePreventionReplacement{
		replacementBase: replacementBase{sourceID: source.ID(), duration: core.WhileOnBattlefield},
		counterType:     e.counterType, incarnation: source.incarnationID, ruleID: e.ruleID,
	})
	return nil
}
func (e *counterPreventionContinuous) cloneEffect() ContinuousEffect {
	clone := *e
	return &clone
}
