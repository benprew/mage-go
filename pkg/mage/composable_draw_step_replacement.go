package mage

import (
	"fmt"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type drawStepLifeRule struct{ amount int }

// SkipDrawStepForLife lets the controller skip their draw step, then gain life.
// The gain occurs first in the next step that occurs. Use this rule with Static.
func SkipDrawStepForLife(amount int) StaticRule {
	if amount < 0 {
		panic("skip draw step: supply a nonnegative life amount")
	}
	return &drawStepLifeRule{amount: amount}
}

func (*drawStepLifeRule) staticRule() {}
func (r *drawStepLifeRule) staticEffects() []ContinuousEffect {
	return []ContinuousEffect{&drawStepLifeContinuous{amount: r.amount, ruleID: uuid.New()}}
}

type drawStepLifeContinuous struct {
	effectSource
	amount int
	ruleID uuid.UUID
}

func (e *drawStepLifeContinuous) Text() string {
	return fmt.Sprintf("you may skip your draw step; if you do, gain %d life", e.amount)
}
func (*drawStepLifeContinuous) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (*drawStepLifeContinuous) GetLayer() Layer       { return LayerAbility }
func (*drawStepLifeContinuous) GetDuration() Duration { return WhileOnBattlefield }
func (e *drawStepLifeContinuous) IsActive(g *Game) bool {
	source := g.FindPermanent(e.sourceID)
	return source != nil && !source.PhasedOut
}
func (e *drawStepLifeContinuous) inspectDefinition() definitionNode {
	return definitionNode{Role: "static-draw-step-replacement", Text: e.Text()}
}
func (e *drawStepLifeContinuous) ApplyContinuous(ctx *EffectContext) error {
	source := ctx.Game.FindPermanent(e.sourceID)
	if source == nil {
		return nil
	}
	ctx.Game.effects.AddCycleReplacement(&drawStepLifeReplacement{
		replacementBase: replacementBase{sourceID: e.sourceID, duration: WhileOnBattlefield},
		incarnation:     source.incarnationID, amount: e.amount, ruleID: e.ruleID,
	})
	return nil
}

func (r *drawStepLifeRule) inspectDefinition() definitionNode {
	return definitionNode{Role: "static-draw-step-replacement", Text: fmt.Sprintf("you may skip your draw step; if you do, gain %d life", r.amount)}
}

func (e *drawStepLifeContinuous) cloneEffect() ContinuousEffect {
	clone := *e
	return &clone
}
