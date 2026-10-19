package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type combatCardinalityRule struct {
	limit  int
	blocks bool
}

// LimitAttackDeclarations limits the creatures declared as attackers in one combat.
func LimitAttackDeclarations(limit int) StaticRule { return combatCardinalityRule{limit: limit} }

// LimitBlockDeclarations counts each creature once when it blocks several attackers.
func LimitBlockDeclarations(limit int) StaticRule {
	return combatCardinalityRule{limit: limit, blocks: true}
}

func (combatCardinalityRule) staticRule() {}
func (r combatCardinalityRule) inspectDefinition() definitionNode {
	n := definitionNode{Role: "static-rule", Text: "combat declaration limit", battlefieldSource: true}
	if r.limit < 0 {
		n.problem = "combat declaration limit must be nonnegative"
	}
	return n
}
func (r combatCardinalityRule) staticEffects() []ContinuousEffect {
	return []ContinuousEffect{&combatCardinalityEffect{funcContinuousEffect: FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil }).(*funcContinuousEffect), rule: r}}
}

type combatCardinalityEffect struct {
	*funcContinuousEffect
	rule combatCardinalityRule
}

func (e *combatCardinalityEffect) inspectDefinition() definitionNode {
	return e.rule.inspectDefinition()
}
func (e *combatCardinalityEffect) cloneEffect() ContinuousEffect {
	clone := *e
	inner := *e.funcContinuousEffect
	clone.funcContinuousEffect = &inner
	return &clone
}

func (g *Game) combatDeclarationLimit(blocks bool) int {
	limit := -1
	for _, effect := range g.effects.effects {
		if !effect.IsActive(g) {
			continue
		}
		inner := effect
		if printed, ok := inner.(*printedAbilityContinuousEffect); ok {
			inner = printed.ContinuousEffect
		}
		if cardinality, ok := inner.(*combatCardinalityEffect); ok && cardinality.rule.blocks == blocks {
			source := g.FindPermanent(effect.SourceID())
			if source != nil && !source.PhasedOut && !source.FaceDown && (limit < 0 || cardinality.rule.limit < limit) {
				limit = cardinality.rule.limit
			}
		}
	}
	return limit
}

func (g *Game) cappedRequiredAttackers(player uuid.UUID, proposed []uuid.UUID, limit int) []uuid.UUID {
	type candidate struct {
		id           uuid.UUID
		requirements int
		preferred    bool
	}
	var candidates []candidate
	seen := make(map[uuid.UUID]bool)
	add := func(p *Permanent, preferred bool) {
		if p == nil || seen[p.ID()] || p.ControllerID() != player || !p.HasType(core.TypeCreature) || !p.CanDeclareAsAttacker(g) || g.combat.IsAttacking(p.ID()) {
			return
		}
		defender := g.GetOpponent(player)
		if defender != nil && g.effects.Rules.IsSanctuaryActive(defender.PlayerID()) && !p.HasKeyword(core.Flying) && !p.HasKeyword(core.Islandwalk) {
			return
		}
		seen[p.ID()] = true
		weight := g.attackRequirementWeight(p)
		if len(g.effects.AttackCosts(p.ID())) > 0 && !preferred {
			return
		}
		candidates = append(candidates, candidate{p.ID(), weight, preferred})
	}
	for _, id := range proposed {
		add(g.FindPermanent(id), true)
	}
	for _, p := range g.zones.battlefield {
		add(p, false)
	}
	slices.SortStableFunc(candidates, func(a, b candidate) int { return b.requirements - a.requirements })
	var result []uuid.UUID
	for _, candidate := range candidates {
		if len(result) == limit {
			break
		}
		if candidate.preferred || candidate.requirements > 0 {
			result = append(result, candidate.id)
		}
	}
	return result
}
