package mage

import (
	"maps"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type outgoingBandSource struct {
	permanent   *Permanent
	amount      int
	information DamageSource
	trample     bool
	deathtouch  bool
	assignment  map[uuid.UUID]int
}

// Other sources can supply lethal damage, but they cannot supply trample.
func outgoingBandLethalRequirements(g *Game, sources []outgoingBandSource, recipients []*Permanent) map[uuid.UUID]int {
	needed := make(map[uuid.UUID]int, len(recipients))
	for _, recipient := range recipients {
		needed[recipient.ID()] = max(recipient.CurrentToughness(g)-recipient.Damage, 0)
	}
	for _, source := range sources {
		for id, assigned := range source.assignment {
			if assigned > 0 && source.deathtouch {
				needed[id] = 0
			} else {
				needed[id] -= min(needed[id], assigned)
			}
		}
	}
	return needed
}

func validOutgoingBandTrample(g *Game, sources []outgoingBandSource, recipients []*Permanent) bool {
	needed := outgoingBandLethalRequirements(g, sources, recipients)
	for _, source := range sources {
		remaining := source.amount
		for _, assigned := range source.assignment {
			remaining -= assigned
		}
		if remaining > 0 {
			for _, amount := range needed {
				if amount > 0 {
					return false
				}
			}
		}
	}
	return true
}

// A fallback still assigns each source separately and retains shared lethal damage.
func defaultOutgoingBandAssignments(g *Game, sources []outgoingBandSource, recipients []*Permanent) {
	for i := range sources {
		sources[i].assignment = make(map[uuid.UUID]int)
	}
	for _, trample := range []bool{false, true} {
		for i := range sources {
			source := &sources[i]
			if source.trample != trample {
				continue
			}
			remaining := source.amount
			needed := outgoingBandLethalRequirements(g, sources, recipients)
			for j, recipient := range recipients {
				lethal := needed[recipient.ID()]
				if source.deathtouch && lethal > 0 {
					lethal = 1
				}
				assigned := min(remaining, lethal)
				if !trample && j == len(recipients)-1 {
					assigned = remaining
				}
				source.assignment[recipient.ID()] = assigned
				remaining -= assigned
			}
		}
	}
}

// Each source must reach the damage pipeline with its own captured information.
func (c *Combat) assignOutgoingBandDamage(g *Game, plan *combatDamagePlan, members, blockers []uuid.UUID, defenderID uuid.UUID, firstStrike bool) {
	var recipients []*Permanent
	defendingBand := g.hasBandingAssignment(blockers)
	for _, id := range blockers {
		blocker := g.FindPermanent(id)
		if blocker == nil || blocker.PhasedOut {
			continue
		}
		recipients = append(recipients, blocker)
	}
	var sources []outgoingBandSource
	for _, id := range members {
		member := g.FindPermanent(id)
		if member == nil || member.PhasedOut || !c.Attackers[id] || !c.assignsDamageInStep(g, member, firstStrike) {
			continue
		}
		if incarnation := c.AttackerIncarnations[id]; incarnation != uuid.Nil && incarnation != member.incarnationID {
			continue
		}
		amount := combatDamageValue(member, g)
		if amount <= 0 {
			continue
		}
		sources = append(sources, outgoingBandSource{permanent: member, amount: amount, information: g.CaptureDamageSource(id), trample: member.HasKeyword(core.Trample), deathtouch: member.HasKeyword(core.Deathtouch)})
	}
	if len(recipients) == 0 {
		if defender := g.GetPlayer(defenderID); defender != nil {
			for _, source := range sources {
				if source.trample {
					plan.assignments = append(plan.assignments, PlayerDamageAssignment(defender, source.amount, source.information))
				}
			}
		}
		return
	}
	needsDefault := false
	for i := range sources {
		source := &sources[i]
		chooser := g.GetPlayer(source.permanent.ControllerID())
		var assignment map[uuid.UUID]int
		if defendingBand {
			chooser = g.GetPlayer(recipients[0].ControllerID())
			switch player := chooser.(type) {
			case BandingDamageAssigner:
				assignment = player.GetBandingDamageAssignment(g, source.permanent, recipients, source.amount)
			case BandingDamageDistributor:
				assignment = player.GetBandingDamageDistribution(recipients)
			}
		} else if assigner, ok := chooser.(CombatDamageAssigner); ok {
			assignment = assigner.GetCombatDamageAssignment(g, source.permanent, recipients, source.amount)
		}
		if validCombatDamageAssignment(assignment, recipients, source.amount, source.trample) {
			source.assignment = maps.Clone(assignment)
		} else if defendingBand && !source.trample {
			source.assignment = chooseBandDamageAssignment(chooser, incomingBandSource{source.permanent, source.amount, source.information}, recipients, nil)
		} else {
			needsDefault = true
		}
	}
	if needsDefault || !validOutgoingBandTrample(g, sources, recipients) {
		defaultOutgoingBandAssignments(g, sources, recipients)
	}
	for _, source := range sources {
		remaining := source.amount
		for _, recipient := range recipients {
			assigned := source.assignment[recipient.ID()]
			if assigned > 0 {
				plan.assignments = append(plan.assignments, PermanentDamageAssignment(recipient, assigned, source.information))
				remaining -= assigned
			}
		}
		if remaining > 0 && source.trample {
			if defender := g.GetPlayer(defenderID); defender != nil {
				plan.assignments = append(plan.assignments, PlayerDamageAssignment(defender, remaining, source.information))
			}
		}
	}
}
