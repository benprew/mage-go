package mage

import (
	"maps"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// AttacksWithoutTappingUntil changes attack tapping without granting vigilance.
// The player is saved. The filter and condition use current state.
func AttacksWithoutTappingUntil(player PlayerRef, filter PermanentFilter, condition StateCondition, duration core.Duration) ResolvingEffect {
	return &attackTappingEffect{player: player, filter: filter, condition: condition, duration: duration}
}

type attackTappingEffect struct {
	player    PlayerRef
	filter    PermanentFilter
	condition StateCondition
	duration  core.Duration
}

func (*attackTappingEffect) Text() string {
	return "attacking does not cause selected creatures to tap"
}
func (*attackTappingEffect) Properties() EffectProperties {
	return EffectProperties{Outcome: OutcomeBenefit}
}
func (e *attackTappingEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", children: []definitionEdge{{"player", e.player}, {"condition", e.condition}}}
	if e.duration != core.EndOfCombat && e.duration != core.EndOfTurn {
		n.problem = "attack tapping requires a fixed combat or turn duration"
	}
	return n
}
func (e *attackTappingEffect) refersToSourcePermanent() bool { return true }
func (e *attackTappingEffect) Apply(ctx *EffectContext) error {
	player, ok := e.player.resolvePlayer(ctx)
	if !ok {
		return nil
	}
	source, ok := This.resolvePermanent(ctx)
	if !ok {
		return nil
	}
	rule := &attackTappingRule{funcContinuousEffect: FuncContinuousEffect(core.LayerAbility, e.duration, func(*Game, uuid.UUID) error { return nil }).(*funcContinuousEffect), player: player, source: capturedPermanent{source.ID(), source.incarnationID}, filter: e.filter, condition: e.condition, state: StateContext{SourceID: source.ID(), SourceIncarnation: source.incarnationID, ControllerID: ctx.Controller, players: maps.Clone(ctx.Game.resolution.playerBindings), cards: maps.Clone(ctx.Game.resolution.cardBindings), permanents: maps.Clone(ctx.Game.resolution.permanentBindings)}}
	rule.SetSourceID(ctx.SourceID)
	ctx.Game.AddContinuousEffect(rule)
	return nil
}

type attackTappingRule struct {
	*funcContinuousEffect
	player    uuid.UUID
	state     StateContext
	source    capturedPermanent
	filter    PermanentFilter
	condition StateCondition
}

func (*attackTappingRule) survivesSourceExit() bool { return true }
func (e *attackTappingRule) cloneEffect() ContinuousEffect {
	clone := *e
	inner := *e.funcContinuousEffect
	clone.funcContinuousEffect = &inner
	clone.state.players = maps.Clone(e.state.players)
	clone.state.cards = maps.Clone(e.state.cards)
	clone.state.permanents = maps.Clone(e.state.permanents)
	return &clone
}
func (g *Game) attackDoesNotTap(p *Permanent) bool {
	if p.HasKeyword(core.Vigilance) {
		return true
	}
	for _, effect := range g.effects.effects {
		rule, ok := effect.(*attackTappingRule)
		if !ok || !rule.IsActive(g) || rule.player != p.ControllerID() || !rule.filter.Match(p, g) {
			continue
		}
		source := g.FindPermanent(rule.source.id)
		if source == nil || source.PhasedOut || source.incarnationID != rule.source.incarnation {
			continue
		}
		if rule.condition.checkState(g, rule.state) {
			return true
		}
	}
	return false
}

func (g *Game) attackTappingExemptions(ids []uuid.UUID) map[capturedPermanent]bool {
	exemptions := make(map[capturedPermanent]bool)
	for _, id := range ids {
		if p := g.FindPermanent(id); p != nil && g.attackDoesNotTap(p) {
			exemptions[capturedPermanent{p.ID(), p.incarnationID}] = true
		}
	}
	return exemptions
}
