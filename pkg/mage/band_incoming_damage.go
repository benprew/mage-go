package mage

import (
	"fmt"
	"maps"
	"math"

	"github.com/google/uuid"
)

type incomingBandSource struct {
	permanent   *Permanent
	amount      int
	information DamageSource
}

func validBandDamageAssignment(assignment map[uuid.UUID]int, recipients []*Permanent, amount int) bool {
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
	return remaining == 0
}

// Keep the requested recipient totals without changing source identity.
func chooseBandDamageAssignment(chooser Player, source incomingBandSource, recipients []*Permanent, quota map[uuid.UUID]int) map[uuid.UUID]int {
	assignment := make(map[uuid.UUID]int, len(recipients))
	remaining := source.amount
	for i, recipient := range recipients {
		minimum, maximum := 0, remaining
		if i == len(recipients)-1 {
			minimum = remaining
		}
		if quota != nil {
			maximum = min(maximum, quota[recipient.ID()])
			later := 0
			for _, other := range recipients[i+1:] {
				later += min(quota[other.ID()], remaining-later)
			}
			minimum = max(minimum, remaining-later)
		}
		assigned := maximum
		if minimum != maximum && chooser != nil {
			reason := fmt.Sprintf("Assign %s's combat damage to %s", source.permanent.Name(), recipient.Name())
			assigned = chooser.ChooseNumber(minimum, maximum, reason)
			if assigned < minimum || assigned > maximum {
				assigned = maximum
			}
		}
		assignment[recipient.ID()] = assigned
		remaining -= assigned
		if quota != nil {
			quota[recipient.ID()] -= assigned
		}
	}
	return assignment
}

// Separate sources let prevention choices and damage results use the correct object.
func (c *Combat) assignIncomingBandDamage(g *Game, plan *combatDamagePlan, members, blockers []uuid.UUID, firstStrike bool) {
	var recipients []*Permanent
	var eligible []uuid.UUID
	for _, id := range members {
		member := g.FindPermanent(id)
		if member == nil || member.PhasedOut || !c.Attackers[id] {
			continue
		}
		if incarnation := c.AttackerIncarnations[id]; incarnation != uuid.Nil && incarnation != member.incarnationID {
			continue
		}
		recipients = append(recipients, member)
		eligible = append(eligible, id)
	}
	banding := g.hasBandingAssignment(eligible)
	if len(recipients) == 0 {
		return
	}
	var sources []incomingBandSource
	total, overflow := 0, false
	for _, id := range blockers {
		blocker := g.FindPermanent(id)
		if blocker == nil || blocker.PhasedOut || !c.assignsDamageInStep(g, blocker, firstStrike) {
			continue
		}
		amount := combatDamageValue(blocker, g)
		if amount <= 0 {
			continue
		}
		sources = append(sources, incomingBandSource{blocker, amount, g.CaptureDamageSource(id)})
		if amount > math.MaxInt-total {
			overflow = true
		} else {
			total += amount
		}
	}
	if len(sources) == 0 {
		return
	}
	attackingPlayer := g.GetPlayer(recipients[0].ControllerID())
	var quota map[uuid.UUID]int
	if _, perSource := attackingPlayer.(BandingDamageAssigner); banding && !perSource && !overflow {
		if legacy, ok := attackingPlayer.(BandingDamageDistributor); ok {
			proposed := legacy.GetBandingDamageDistribution(recipients)
			if validBandDamageAssignment(proposed, recipients, total) {
				quota = make(map[uuid.UUID]int, len(proposed))
				maps.Copy(quota, proposed)
			}
		}
	}
	for _, source := range sources {
		chooser := attackingPlayer
		if !banding {
			chooser = g.GetPlayer(source.permanent.ControllerID())
		}
		var assignment map[uuid.UUID]int
		if assigner, ok := chooser.(BandingDamageAssigner); ok {
			assignment = assigner.GetBandingDamageAssignment(g, source.permanent, recipients, source.amount)
		}
		if !validBandDamageAssignment(assignment, recipients, source.amount) {
			assignment = chooseBandDamageAssignment(chooser, source, recipients, quota)
		}
		for _, recipient := range recipients {
			if amount := assignment[recipient.ID()]; amount > 0 {
				plan.assignments = append(plan.assignments, PermanentDamageAssignment(recipient, amount, source.information))
			}
		}
	}
}
