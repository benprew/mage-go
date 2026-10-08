package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// ChosenPlayerOf reads the entry choice of the current permanent incarnation.
// An absent permanent or choice does not supply a player.
func ChosenPlayerOf(subject PermanentRef) PlayerRef { return chosenPermanentPlayerRef{subject} }

type chosenPermanentPlayerRef struct{ subject PermanentRef }

func (chosenPermanentPlayerRef) playerRef()       {}
func (chosenPermanentPlayerRef) damageRecipient() {}
func (r chosenPermanentPlayerRef) resolveRecipient(ctx *EffectContext) (uuid.UUID, bool) {
	return r.resolvePlayer(ctx)
}
func (r chosenPermanentPlayerRef) resolvePlayer(ctx *EffectContext) (uuid.UUID, bool) {
	p, ok := r.subject.resolvePermanent(ctx)
	if !ok {
		return uuid.Nil, false
	}
	return p.ChosenPlayer, ctx.Game.GetPlayer(p.ChosenPlayer) != nil
}
func (r chosenPermanentPlayerRef) inspectDefinition() definitionNode {
	return definitionNode{Role: "player-ref", Text: "entry-chosen player", children: []definitionEdge{{"subject", r.subject}}}
}

// SetMaximumHandSizeStatic sets a live player's maximum hand size.
// Rule effects use timestamp order after the object layers.
func SetMaximumHandSizeStatic(subject PlayerRef, size int) ContinuousEffect {
	return &maximumHandSizeEffect{subject: subject, size: size}
}

// NoMaximumHandSizeStatic removes a live player's maximum hand size.
func NoMaximumHandSizeStatic(subject PlayerRef) ContinuousEffect {
	return &maximumHandSizeEffect{subject: subject, unlimited: true}
}

type maximumHandSizeEffect struct {
	effectSource
	subject   PlayerRef
	size      int
	unlimited bool
}

func (*maximumHandSizeEffect) GetLayer() core.Layer         { return core.LayerAbility }
func (*maximumHandSizeEffect) GetDuration() core.Duration   { return core.WhileOnBattlefield }
func (e *maximumHandSizeEffect) IsActive(g *Game) bool      { return g.FindPermanent(e.SourceID()) != nil }
func (*maximumHandSizeEffect) Text() string                 { return "set maximum hand size" }
func (*maximumHandSizeEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *maximumHandSizeEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "maximum-hand-size", Text: e.Text(), battlefieldSource: true, children: []definitionEdge{{"subject", e.subject}}}
	if !e.unlimited && e.size < 0 {
		n.problem = "supply a nonnegative maximum hand size"
	}
	return n
}
func (*maximumHandSizeEffect) ApplyContinuous(*EffectContext) error { return nil }

func (e *maximumHandSizeEffect) cloneEffect() ContinuousEffect {
	clone := *e
	return &clone
}

func (g *Game) maximumHandSizeWithStaticRules(playerID uuid.UUID) int {
	size := g.effects.Rules.MaxHandSize(playerID)
	// CR 613.11 reads rule effects after the object layers, in timestamp order.
	for _, effect := range g.effects.effects {
		inner := effect
		if printed, ok := inner.(*printedAbilityContinuousEffect); ok {
			inner = printed.ContinuousEffect
		}
		rule, ok := inner.(*maximumHandSizeEffect)
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
			size = rule.size
			if rule.unlimited {
				size = -1
			}
		}
	}
	return size
}
