package mage

import "github.com/google/uuid"

func validCombatDamageAssignment(assignment map[uuid.UUID]int, recipients []*Permanent, amount int, trample bool) bool {
	if assignment == nil {
		return false
	}
	legal := make(map[uuid.UUID]bool, len(recipients))
	for _, recipient := range recipients {
		legal[recipient.ID()] = true
	}
	remaining := amount
	for id, assigned := range assignment {
		if !legal[id] || assigned < 0 || assigned > remaining {
			return false
		}
		remaining -= assigned
	}
	return remaining == 0 || trample
}

func liveNormalBlockers(g *Game, group *CombatGroup) []*Permanent {
	blockers := make([]*Permanent, 0, len(group.BlockerIDs))
	for i, id := range group.BlockerIDs {
		blocker := g.FindPermanent(id)
		if blocker == nil || blocker.PhasedOut {
			continue
		}
		if i < len(group.BlockerIncarnations) {
			incarnation := group.BlockerIncarnations[i]
			if incarnation != uuid.Nil && incarnation != blocker.incarnationID {
				continue
			}
		}
		blockers = append(blockers, blocker)
	}
	return blockers
}

// Nontrample assignments have no lethal order, but must use the complete amount.
func (c *Combat) doNontrampleBlockedDamage(g *Game, plan *combatDamagePlan, attacker *Permanent, group *CombatGroup, firstStrike bool) {
	if attacker.PhasedOut || !c.Attackers[attacker.ID()] {
		return
	}
	if incarnation := c.AttackerIncarnations[attacker.ID()]; incarnation != uuid.Nil && incarnation != attacker.incarnationID {
		return
	}
	blockers := liveNormalBlockers(g, group)
	if c.DealsDamageInStep(attacker, firstStrike) {
		amount := combatDamageValue(attacker, g)
		if amount > 0 && len(blockers) > 0 {
			source := g.CaptureDamageSource(attacker.ID())
			var assignment map[uuid.UUID]int
			if assigner, ok := g.GetPlayer(attacker.ControllerID()).(CombatDamageAssigner); ok && len(blockers) > 1 {
				assignment = assigner.GetCombatDamageAssignment(g, attacker, blockers, amount)
			}
			if validCombatDamageAssignment(assignment, blockers, amount, false) {
				for _, blocker := range blockers {
					if assigned := assignment[blocker.ID()]; assigned > 0 {
						plan.assignments = append(plan.assignments, PermanentDamageAssignment(blocker, assigned, source))
					}
				}
			} else {
				dealDefaultBlockedCombatDamage(g, plan, attacker, blockers, amount, false)
			}
		}
	}
	for _, blocker := range blockers {
		if c.DealsDamageInStep(blocker, firstStrike) {
			plan.permanent(g, attacker, combatDamageValue(blocker, g), blocker.ID())
		}
	}
}
