package mage

import (
	"fmt"

	. "github.com/benprew/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

type randomCounterDistributionEffect struct {
	counterType CounterType
	total       ValueSource
}

// RandomCounterDistribution creates an effect whose total counters are
// randomly distributed among its targets as the stack object is created.
// Every target receives at least one counter; each remaining counter is
// independently assigned to a uniformly random target.
func RandomCounterDistribution(counterType CounterType, total ValueSource) Effect {
	return &randomCounterDistributionEffect{counterType: counterType, total: total}
}

func (e *randomCounterDistributionEffect) Text() string {
	return fmt.Sprintf("randomly distribute %s %s counters among the targets", e.total.Text(), e.counterType)
}

func (*randomCounterDistributionEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}

func (e *randomCounterDistributionEffect) Apply(ctx *EffectContext) error {
	for _, targetID := range ctx.Targets {
		amount := ctx.CounterDistribution[targetID]
		if amount <= 0 {
			continue
		}
		perm := ctx.Game.FindPermanent(targetID)
		if perm == nil {
			continue
		}
		ctx.Game.AddCountersWithReplacement(perm, e.counterType, amount, ctx.SourceID, false)
	}
	return nil
}

func (g *Game) chooseRandomCounterDistribution(obj *StackObject) {
	if obj == nil || obj.CounterDistribution != nil || len(obj.Targets) == 0 {
		return
	}
	for _, effect := range obj.Effects {
		distribution, ok := effect.(*randomCounterDistributionEffect)
		if !ok {
			continue
		}
		previousX := g.currentX
		g.currentX = obj.XValue
		total := distribution.total.Resolve(g, obj.SourceID, obj.Controller, obj.Targets)
		g.currentX = previousX
		if total < len(obj.Targets) {
			return
		}
		obj.CounterDistribution = make(map[uuid.UUID]int, len(obj.Targets))
		for _, targetID := range obj.Targets {
			obj.CounterDistribution[targetID]++
		}
		for range total - len(obj.Targets) {
			targetID := obj.Targets[g.RandIntn(len(obj.Targets))]
			obj.CounterDistribution[targetID]++
		}
		return
	}
}
