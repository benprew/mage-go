package heuristic

import (
	"uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/interactive/eval"
)

func (s *Strategy) removalCastValue(g *mage.Game, playerID uuid.UUID, card *mage.Card, targets []uuid.UUID, x int) (int, bool) {
	if eval.RemovalCardValue(g, g.GetPlayer(playerID), card) <= 0 {
		return 0, false
	}
	after := simulateSpellCast(g, playerID, card, targets, x)
	if after == nil {
		return 0, false
	}
	before := g.Clone()
	before.ResolveStack()
	before.CheckStateBasedActions()
	resolveDeclaredCombat(before)
	resolveDeclaredCombat(after)
	// The lost hand value is the cost of spending this answer now.
	gain := s.castPositionValue(after, playerID) - s.castPositionValue(before, playerID)
	return roundCastValue(gain), true
}

func resolveDeclaredCombat(g *mage.Game) {
	if g.IsGameOver() || len(g.CombatGroups()) == 0 {
		return
	}
	switch g.GetStep() {
	case core.DeclareBlockers:
		g.RunStepWithPriority(core.FirstStrikeDamage)
		if !g.IsGameOver() {
			g.RunStepWithPriority(core.CombatDamage)
		}
	case core.FirstStrikeDamage:
		g.RunStepWithPriority(core.CombatDamage)
	}
}
