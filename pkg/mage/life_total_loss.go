package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// PreventLifeTotalLossStatic stops loss from a nonpositive life total.
// Other causes of game loss still apply.
func PreventLifeTotalLossStatic(subject PlayerRef) ContinuousEffect {
	return &lifeTotalLossExemption{subject: subject}
}

type lifeTotalLossExemption struct {
	effectSource
	subject PlayerRef
}

func (*lifeTotalLossExemption) GetLayer() core.Layer                 { return core.LayerAbility }
func (*lifeTotalLossExemption) GetDuration() core.Duration           { return core.WhileOnBattlefield }
func (e *lifeTotalLossExemption) IsActive(g *Game) bool              { return g.FindPermanent(e.SourceID()) != nil }
func (*lifeTotalLossExemption) Text() string                         { return "do not lose from a nonpositive life total" }
func (*lifeTotalLossExemption) Properties() EffectProperties         { return EffectProperties{} }
func (*lifeTotalLossExemption) ApplyContinuous(*EffectContext) error { return nil }
func (e *lifeTotalLossExemption) inspectDefinition() definitionNode {
	return definitionNode{Role: "life-total-loss-exemption", Text: e.Text(), battlefieldSource: true, children: []definitionEdge{{"subject", e.subject}}}
}
func (e *lifeTotalLossExemption) cloneEffect() ContinuousEffect { clone := *e; return &clone }

// PreventsLifeTotalLoss reports whether a live static rule stops life-total loss.
// It does not change player loss state.
func (g *Game) PreventsLifeTotalLoss(playerID uuid.UUID) bool {
	for _, effect := range g.effects.effects {
		inner := effect
		if printed, ok := inner.(*printedAbilityContinuousEffect); ok {
			inner = printed.ContinuousEffect
		}
		rule, ok := inner.(*lifeTotalLossExemption)
		if !ok || !effect.IsActive(g) {
			continue
		}
		source := g.FindPermanent(rule.SourceID())
		if source == nil {
			continue
		}
		frame := &EffectContext{Game: g, SourceID: source.ID(), Controller: source.ControllerID(), sourceIncarnation: source.incarnationID, sourceCaptured: true}
		id, ok := rule.subject.resolvePlayer(frame)
		if ok && id == playerID {
			return true
		}
	}
	return false
}
