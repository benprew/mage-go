package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// CouldntAttackThisTurn reads attack opportunities for the exact battlefield visit.
// Optional attack costs do not require a player to enable an attack.
var CouldntAttackThisTurn = NewPermanentFilter("could not attack this turn", func(p *Permanent, g *Game) bool {
	return p != nil && g != nil && !g.trackers.Turn.attackOpportunities[capturedPermanent{p.ID(), p.incarnationID}] && !g.HasAttackedThisTurnIncarnation(p.ID(), p.incarnationID)
})

func (g *Game) attackRequirementWeight(p *Permanent) int {
	weight := blockingAttributeCount(p, core.AttrMustAttack)
	for _, effect := range g.effects.effects {
		if rule, ok := effect.(*turnAttackRule); ok && rule.require && rule.IsActive(g) && rule.matches(g, p) {
			weight++
		}
	}
	return weight
}

// recordAttackOpportunities runs before choices, payments, taps and attack events.
// A declaration limit can exclude a creature even when its own restrictions pass.
func (g *Game) recordAttackOpportunities(player uuid.UUID) {
	defender := g.GetOpponent(player)
	if defender == nil {
		return
	}
	var candidates []*Permanent
	var weights []int
	for _, p := range g.zones.battlefield {
		if p.ControllerID() != player || !p.HasType(core.TypeCreature) || !p.CanDeclareAsAttacker(g) || len(g.effects.AttackCosts(p.ID())) > 0 {
			continue
		}
		if g.effects.Rules.IsSanctuaryActive(defender.PlayerID()) && !p.HasKeyword(core.Flying) && !p.HasKeyword(core.Islandwalk) {
			continue
		}
		candidates = append(candidates, p)
		weights = append(weights, g.attackRequirementWeight(p))
	}
	limit := g.combatDeclarationLimit(false)
	if limit == 0 {
		return
	}
	threshold := 0
	if limit > 0 && limit <= len(weights) {
		slices.Sort(weights)
		threshold = weights[len(weights)-limit]
	}
	if g.trackers.Turn.attackOpportunities == nil {
		g.trackers.Turn.attackOpportunities = make(map[capturedPermanent]bool)
	}
	for _, p := range candidates {
		if g.attackRequirementWeight(p) >= threshold {
			g.trackers.Turn.attackOpportunities[capturedPermanent{p.ID(), p.incarnationID}] = true
		}
	}
}
