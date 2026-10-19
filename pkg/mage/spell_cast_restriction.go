package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type spellCastRestrictionRule struct{ filter CardFilter }

// PreventSpellCasts prevents each player from casting matching spells.
// Direct battlefield entry does not use this rule.
func PreventSpellCasts(filter CardFilter) StaticRule { return spellCastRestrictionRule{filter: filter} }

func (spellCastRestrictionRule) staticRule() {}
func (r spellCastRestrictionRule) inspectDefinition() definitionNode {
	node := definitionNode{Role: "static-rule", Text: "players cannot cast matching spells", battlefieldSource: true}
	if r.filter.IsZero() {
		node.problem = "spell cast restriction requires a card filter"
	}
	return node
}
func (r spellCastRestrictionRule) staticEffects() []ContinuousEffect {
	return []ContinuousEffect{&spellCastRestrictionEffect{
		funcContinuousEffect: FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(*Game, uuid.UUID) error { return nil }).(*funcContinuousEffect),
		rule:                 r,
	}}
}

type spellCastRestrictionEffect struct {
	*funcContinuousEffect
	rule spellCastRestrictionRule
}

func (e *spellCastRestrictionEffect) inspectDefinition() definitionNode {
	return e.rule.inspectDefinition()
}
func (e *spellCastRestrictionEffect) cloneEffect() ContinuousEffect {
	clone := *e
	inner := *e.funcContinuousEffect
	clone.funcContinuousEffect = &inner
	return &clone
}

// IsSpellCastProhibited checks current restrictions without changing game state.
// Timing and zone permissions require separate checks.
func (g *Game) IsSpellCastProhibited(playerID uuid.UUID, card Card) bool {
	if card == nil || card.HasType(core.TypeLand) {
		return false
	}
	if g.isHandVisitPlayLocked(playerID, card.ID()) {
		return true
	}
	if g.effects.Rules.PlayerCantCastSpells(playerID) || g.effects.Rules.IsCardExpansionBlocked(card.Name()) {
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
		if rule, ok := inner.(*spellCastRestrictionEffect); ok && rule.rule.filter.Match(card) {
			source := g.FindPermanent(effect.SourceID())
			if source != nil && !source.FaceDown {
				return true
			}
		}
	}
	return false
}
