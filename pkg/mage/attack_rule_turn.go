package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// CantAttackMatchingThisTurn checks current characteristics until cleanup.
// The restriction remains separate from the creatures' abilities.
func CantAttackMatchingThisTurn(filter PermanentFilter) ResolvingEffect {
	return &turnAttackRuleEffect{filter: filter}
}

// MustAttackControlledThisTurn requires matching creatures to attack if able.
// The player is saved at resolution; creature membership remains current.
func MustAttackControlledThisTurn(player PlayerRef, filter PermanentFilter) ResolvingEffect {
	return &turnAttackRuleEffect{filter: filter, player: player, require: true}
}

// MustAttackThisTurn requires the saved permanent to attack if able until cleanup.
// Control changes keep the rule. A returned permanent cannot inherit it.
func MustAttackThisTurn(subject PermanentRef) ResolvingEffect {
	return &singularTurnAttackRuleEffect{subject: subject}
}

type singularTurnAttackRuleEffect struct{ subject PermanentRef }

func (*singularTurnAttackRuleEffect) Text() string {
	return "the saved permanent attacks if able this turn"
}
func (*singularTurnAttackRuleEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *singularTurnAttackRuleEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "resolving-effect", Text: e.Text(), children: []definitionEdge{{"subject", e.subject}}}
}
func (e *singularTurnAttackRuleEffect) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.subject)
}
func (e *singularTurnAttackRuleEffect) Apply(ctx *EffectContext) error {
	p, ok := e.subject.resolvePermanent(ctx)
	if !ok {
		return nil
	}
	rule := &turnAttackRule{
		funcContinuousEffect: FuncContinuousEffect(core.LayerAbility, core.EndOfTurn, func(*Game, uuid.UUID) error { return nil }).(*funcContinuousEffect),
		filter:               IsCreature, require: true, subject: capturedPermanent{p.ID(), p.incarnationID},
	}
	rule.SetSourceID(ctx.SourceID)
	ctx.Game.AddContinuousEffect(rule)
	return nil
}

type turnAttackRuleEffect struct {
	filter  PermanentFilter
	player  PlayerRef
	require bool
}

func (e *turnAttackRuleEffect) Text() string {
	if e.require {
		return "matching creatures attack if able this turn"
	}
	return "matching creatures cannot attack this turn"
}
func (*turnAttackRuleEffect) Properties() EffectProperties { return EffectProperties{} }
func (e *turnAttackRuleEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "resolving-effect", Text: e.Text()}
	if e.require {
		n.children = []definitionEdge{{"player", e.player}}
	}
	return n
}
func (e *turnAttackRuleEffect) Apply(ctx *EffectContext) error {
	controller := uuid.Nil
	if e.require {
		var ok bool
		controller, ok = e.player.resolvePlayer(ctx)
		if !ok {
			return nil
		}
	}
	rule := &turnAttackRule{
		funcContinuousEffect: FuncContinuousEffect(core.LayerAbility, core.EndOfTurn, func(*Game, uuid.UUID) error { return nil }).(*funcContinuousEffect),
		filter:               e.filter, controller: controller, require: e.require,
	}
	rule.SetSourceID(ctx.SourceID)
	ctx.Game.AddContinuousEffect(rule)
	return nil
}

type turnAttackRule struct {
	*funcContinuousEffect
	filter     PermanentFilter
	controller uuid.UUID
	require    bool
	subject    capturedPermanent
}

func (*turnAttackRule) survivesSourceExit() bool { return true }
func (e *turnAttackRule) cloneEffect() ContinuousEffect {
	clone := *e
	inner := *e.funcContinuousEffect
	clone.funcContinuousEffect = &inner
	return &clone
}
func (e *turnAttackRule) matches(g *Game, p *Permanent) bool {
	return !p.PhasedOut && (e.subject.id == uuid.Nil || e.subject.id == p.ID() && e.subject.incarnation == p.incarnationID) && (e.controller == uuid.Nil || p.ControllerID() == e.controller) && e.filter.Match(p, g)
}
func (g *Game) hasTurnAttackRule(p *Permanent, require bool) bool {
	for _, effect := range g.effects.effects {
		if rule, ok := effect.(*turnAttackRule); ok && rule.require == require && rule.IsActive(g) && rule.matches(g, p) {
			return true
		}
	}
	return false
}
func (g *Game) completeRequiredAttackers(player uuid.UUID, proposed []uuid.UUID, includePrinted bool) []uuid.UUID {
	result := append([]uuid.UUID(nil), proposed...)
	declared := make(map[uuid.UUID]bool, len(result))
	for _, id := range result {
		declared[id] = true
	}
	for _, p := range g.zones.battlefield {
		if p.ControllerID() == player && !declared[p.ID()] && (includePrinted && p.HasAttr(core.AttrMustAttack) || g.hasTurnAttackRule(p, true)) && p.CanDeclareAsAttacker(g) {
			if g.hasTurnAttackRule(p, true) && (!includePrinted || !p.HasAttr(core.AttrMustAttack)) && len(g.effects.AttackCosts(p.ID())) > 0 {
				continue
			}
			result = append(result, p.ID())
		}
	}
	return result
}
