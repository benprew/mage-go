package mage

import (
	"log"
	"maps"
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// ReplaceNextDrawThisTurn replaces one saved player's next draw until cleanup.
// The body keeps the source visit and bindings from installation.
func ReplaceNextDrawThisTurn(player PlayerRef, body ResolvingEffect) ResolvingEffect {
	return &futureDrawInstallation{player: player, body: body}
}

type futureDrawInstallation struct {
	player PlayerRef
	body   ResolvingEffect
}

func (*futureDrawInstallation) Text() string                 { return "replace the next card draw this turn" }
func (*futureDrawInstallation) Properties() EffectProperties { return EffectProperties{} }
func (e *futureDrawInstallation) inspectDefinition() definitionNode {
	n := definitionNode{Role: "future-draw-replacement", Text: e.Text(), isolatedBindings: true, children: []definitionEdge{{"player", e.player}, {"body", e.body}}}
	if nilDefinition(e.player) || nilDefinition(e.body) {
		n.problem = "supply a player and replacement body"
	}
	return n
}
func (e *futureDrawInstallation) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || effectReadsSourcePermanent(e.body)
}
func (e *futureDrawInstallation) Apply(ctx *EffectContext) error {
	player, ok := e.player.resolvePlayer(ctx)
	if !ok || ctx.Game.GetPlayer(player) == nil {
		return nil
	}
	saved := cloneFutureDrawContext(*ctx)
	saved.Game = nil
	if !saved.sourceCaptured {
		if source := ctx.Game.FindPermanentIncludingPhased(ctx.SourceID); source != nil {
			saved.sourceIncarnation = source.incarnationID
		}
	}
	saved.sourceCaptured = true
	frame := ctx.Game.resolution.Clone()
	frame.manaTracking = false
	frame.resolvingStackID = uuid.Nil
	frame.resolvingCard = nil
	frame.resolvingCastContext = ctx.Game.resolution.resolvingCastContext.clone()
	ctx.Game.effects.AddReplacement(&futureDrawReplacement{
		replacementBase: replacementBase{sourceID: ctx.SourceID, duration: core.EndOfTurn},
		player:          player, body: e.body, context: saved, frame: frame,
	})
	return nil
}

func cloneFutureDrawContext(ctx EffectContext) EffectContext {
	ctx.Targets = slices.Clone(ctx.Targets)
	ctx.targetBindings = cloneDelayedTargets(ctx.targetBindings)
	ctx.Vars = maps.Clone(ctx.Vars)
	ctx.DamageDistribution = maps.Clone(ctx.DamageDistribution)
	ctx.CounterDistribution = maps.Clone(ctx.CounterDistribution)
	return ctx
}

type futureDrawReplacement struct {
	replacementBase
	player   uuid.UUID
	body     ResolvingEffect
	context  EffectContext
	frame    ResolutionState
	consumed bool
}

func (r *futureDrawReplacement) Text() string             { return "replace the next card draw this turn" }
func (*futureDrawReplacement) survivesSourceExit() bool   { return true }
func (r *futureDrawReplacement) IsActive(GameReader) bool { return !r.consumed }
func (r *futureDrawReplacement) Matches(action Action, _ GameReader) bool {
	draw, ok := action.(*DrawCardAction)
	return ok && !r.consumed && draw.PlayerID() == r.player
}
func (r *futureDrawReplacement) Replace(action Action, g *Game) Action {
	if !r.IsActive(g) || !r.Matches(action, g) {
		return action
	}
	r.consumed = true
	outer := g.resolution
	defer func() { g.resolution = outer }()
	g.resolution = r.frame.Clone()
	g.resolution.resolvingCastContext = r.frame.resolvingCastContext.clone()
	g.resolution.drawReplacementRules = maps.Clone(action.(*DrawCardAction).excludedRules)
	ctx := cloneFutureDrawContext(r.context)
	ctx.Game = g
	if err := r.body.Apply(&ctx); err != nil {
		log.Printf("Future draw replacement failed: %v", err)
	}
	return nil
}
func (r *futureDrawReplacement) Clone() ReplacementEffect {
	c := *r
	c.context = cloneFutureDrawContext(r.context)
	c.frame = r.frame.Clone()
	c.frame.resolvingCastContext = r.frame.resolvingCastContext.clone()
	return &c
}
