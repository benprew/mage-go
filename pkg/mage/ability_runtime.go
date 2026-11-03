package mage

import "uuid"

type abilityActivationKey struct {
	source, ability uuid.UUID
	grant           uuid.UUID
	occurrence      int
}

func (g *Game) activationKey(source, ability uuid.UUID, indices ...int) abilityActivationKey {
	occurrence := 0
	var grant uuid.UUID
	if permanent := g.FindPermanent(source); permanent != nil {
		if len(indices) > 0 && indices[0] >= 0 && indices[0] < len(permanent.RuntimeAbilities) {
			if granted, ok := permanent.RuntimeAbilities[indices[0]].(*grantedByEffect); ok {
				grant = granted.grantID
			}
			for i, previous := range permanent.RuntimeAbilities {
				if i >= indices[0] {
					break
				}
				var previousGrant uuid.UUID
				if granted, ok := previous.(*grantedByEffect); ok {
					previousGrant = granted.grantID
				}
				if previous.AbilityID() == ability && previousGrant == grant {
					occurrence++
				}
			}
		}
		source = permanent.incarnationID
	} else if incarnation, ok := g.abilityZoneIncarnations[source]; ok {
		source = incarnation
	}
	return abilityActivationKey{source: source, ability: ability, grant: grant, occurrence: occurrence}
}

func (g *Game) activationCount(source, ability uuid.UUID, indices ...int) int {
	return g.abilityActivations[g.activationKey(source, ability, indices...)]
}

func (g *Game) markAbilityActivated(source, ability uuid.UUID, indices ...int) {
	if g.abilityActivations == nil {
		g.abilityActivations = make(map[abilityActivationKey]int)
	}
	g.abilityActivations[g.activationKey(source, ability, indices...)]++
}
