package mage

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// GrantEachBandsWithOther grants a combat ability with the given quality in layer 6.
func GrantEachBandsWithOther(subjects PermanentSet, quality PermanentFilter) ContinuousModification {
	return &layerModification{subjects: subjects, bandsQuality: &quality, layer: core.LayerAbility}
}

func (g *Game) bandsQualities(p *Permanent) []PermanentFilter {
	return g.effects.bandsWithOther[capturedPermanent{p.ID(), p.incarnationID}]
}

func (g *Game) liveCombatMembers(ids []uuid.UUID) []*Permanent {
	var result []*Permanent
	for _, id := range ids {
		p := g.FindPermanent(id)
		if p != nil && !p.PhasedOut {
			result = append(result, p)
		}
	}
	return result
}

// Formation does not preserve damage assignment permission after ability loss.
func (g *Game) hasBandingAssignment(ids []uuid.UUID) bool {
	members := g.liveCombatMembers(ids)
	for _, holder := range members {
		if holder.HasKeyword(core.Banding) {
			return true
		}
		for _, quality := range g.bandsQualities(holder) {
			if !quality.Match(holder, g) {
				continue
			}
			for _, other := range members {
				if other.ID() != holder.ID() && quality.Match(other, g) {
					return true
				}
			}
		}
	}
	return false
}

func (g *Game) isValidBand(ids []uuid.UUID) bool {
	if len(ids) == 0 {
		return false
	}
	members := g.liveCombatMembers(ids)
	if len(members) != len(ids) {
		return false
	}
	seen := make(map[uuid.UUID]bool)
	defender, controller := uuid.Nil, uuid.Nil
	ordinary, without := 0, 0
	for _, p := range members {
		group := g.combat.GroupFor(p.ID())
		incarnation := g.combat.AttackerIncarnations[p.ID()]
		if seen[p.ID()] || !g.combat.IsAttacking(p.ID()) || g.combat.IsInBand(p.ID()) || group == nil || group.Blocked || group.Unblocked || (incarnation != uuid.Nil && incarnation != p.incarnationID) {
			return false
		}
		seen[p.ID()] = true
		if (defender != uuid.Nil && defender != group.DefenderID) || (controller != uuid.Nil && controller != p.ControllerID()) {
			return false
		}
		defender, controller = group.DefenderID, p.ControllerID()
		if p.HasKeyword(core.Banding) {
			ordinary++
		} else {
			without++
		}
	}
	if ordinary > 0 && without <= 1 {
		return true
	}
	for _, holder := range members {
		for _, quality := range g.bandsQualities(holder) {
			matches := true
			for _, p := range members {
				matches = matches && quality.Match(p, g)
			}
			if matches {
				return true
			}
		}
	}
	return false
}

// FormAttackingBand checks current attackers before it saves their band.
func (g *Game) FormAttackingBand(ids []uuid.UUID) error {
	if !g.isValidBand(ids) {
		return fmt.Errorf("invalid attacking band")
	}
	g.combat.AddBand(append([]uuid.UUID(nil), ids...))
	return nil
}

func (c *Combat) needsSharedBandAssignment(g *Game) bool {
	blockers := make(map[uuid.UUID]map[uuid.UUID]bool)
	for _, group := range c.Groups {
		ids := append([]uuid.UUID{group.AttackerID}, group.BlockerIDs...)
		for _, p := range g.liveCombatMembers(ids) {
			if len(g.bandsQualities(p)) > 0 {
				return true
			}
		}
		leader := group.AttackerID
		if c.IsInBand(leader) {
			leader = c.Bands[leader][0]
		}
		for _, id := range group.BlockerIDs {
			if blockers[id] == nil {
				blockers[id] = make(map[uuid.UUID]bool)
			}
			blockers[id][leader] = true
			if len(blockers[id]) > 1 {
				return true
			}
		}
	}
	return false
}

// One blocker has one damage amount even when it blocks several attackers.
func (c *Combat) resolveSharedBandDamage(g *Game, firstStrike bool) {
	plan := &combatDamagePlan{}
	processed := make(map[uuid.UUID]bool)
	incoming := make(map[uuid.UUID][]uuid.UUID)
	var blockerOrder []uuid.UUID
	for _, group := range c.Groups {
		attacker := g.FindPermanent(group.AttackerID)
		if attacker == nil || attacker.PhasedOut || !c.Attackers[attacker.ID()] {
			continue
		}
		if incarnation := c.AttackerIncarnations[attacker.ID()]; incarnation != uuid.Nil && incarnation != attacker.incarnationID {
			continue
		}
		for _, blocker := range liveNormalBlockers(g, group) {
			if len(incoming[blocker.ID()]) == 0 {
				blockerOrder = append(blockerOrder, blocker.ID())
			}
			incoming[blocker.ID()] = append(incoming[blocker.ID()], attacker.ID())
		}
		members := []uuid.UUID{attacker.ID()}
		if c.IsInBand(attacker.ID()) {
			members = c.Bands[attacker.ID()]
		}
		leader := members[0]
		if processed[leader] {
			continue
		}
		processed[leader] = true
		blocked := false
		var blockers []uuid.UUID
		seen := make(map[uuid.UUID]bool)
		for _, id := range members {
			if memberGroup := c.GroupFor(id); memberGroup != nil {
				blocked = blocked || memberGroup.Blocked
				for _, p := range liveNormalBlockers(g, memberGroup) {
					if !seen[p.ID()] {
						seen[p.ID()] = true
						blockers = append(blockers, p.ID())
					}
				}
			}
		}
		if blocked {
			c.assignOutgoingBandDamage(g, plan, members, blockers, group.DefenderID, firstStrike)
		} else if defender := g.GetPlayer(group.DefenderID); defender != nil {
			for _, id := range members {
				p := g.FindPermanent(id)
				if p != nil && !p.PhasedOut && c.Attackers[id] && c.assignsDamageInStep(g, p, firstStrike) {
					plan.player(g, defender, combatDamageValue(p, g), id)
				}
			}
		}
	}
	for _, id := range blockerOrder {
		c.assignIncomingBandDamage(g, plan, incoming[id], []uuid.UUID{id}, firstStrike)
	}
	g.DealSimultaneousDamage(plan.assignments...)
	g.CheckStateBasedActions()
}
