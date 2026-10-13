package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type landPlayRestrictionRule struct{ filter CardFilter }

// PreventLandPlays prevents each player from playing matching land cards.
// Direct battlefield entry does not use this rule.
func PreventLandPlays(filter CardFilter) StaticRule { return landPlayRestrictionRule{filter: filter} }

func (landPlayRestrictionRule) staticRule() {}
func (r landPlayRestrictionRule) inspectDefinition() definitionNode {
	node := definitionNode{Role: "static-rule", Text: "players cannot play matching lands", battlefieldSource: true}
	if r.filter.IsZero() {
		node.problem = "land play restriction requires a card filter"
	}
	return node
}
func (r landPlayRestrictionRule) staticEffects() []ContinuousEffect {
	return []ContinuousEffect{&landPlayRestrictionEffect{
		funcContinuousEffect: FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil }).(*funcContinuousEffect),
		rule:                 r,
	}}
}

type landPlayRestrictionEffect struct {
	*funcContinuousEffect
	rule landPlayRestrictionRule
}

func (e *landPlayRestrictionEffect) inspectDefinition() definitionNode {
	return e.rule.inspectDefinition()
}
func (e *landPlayRestrictionEffect) cloneEffect() ContinuousEffect {
	clone := *e
	inner := *e.funcContinuousEffect
	clone.funcContinuousEffect = &inner
	return &clone
}

// IsLandPlayProhibited checks current restrictions without changing game state.
// Timing, zones and the land play limit require separate checks.
func (g *Game) IsLandPlayProhibited(card Card) bool {
	if card == nil || !card.HasType(core.TypeLand) {
		return false
	}
	if g.effects.Rules.CantPlayLands() || g.effects.Rules.IsCardExpansionBlocked(card.Name()) {
		return true
	}
	for _, effect := range g.effects.effects {
		if !effect.IsActive(g) {
			continue
		}
		inner := effect
		if printed, ok := inner.(*printedAbilityContinuousEffect); ok {
			inner = printed.ContinuousEffect
		}
		if rule, ok := inner.(*landPlayRestrictionEffect); ok && rule.rule.filter.Match(card) {
			source := g.FindPermanent(effect.SourceID())
			if source != nil && !source.FaceDown {
				return true
			}
		}
	}
	return false
}
