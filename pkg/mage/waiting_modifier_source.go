package mage

import "github.com/google/uuid"

// WaitingModifierPermanentSources reads explicit sources of active modifiers.
// Queries do not call functions shared with cloned games.
func (g *Game) WaitingModifierPermanentSources() []DamageSource {
	var identities []capturedPermanent
	addRegistered := func(id, incarnation uuid.UUID, hasFunction bool) {
		if !hasFunction || incarnation == uuid.Nil {
			return
		}
		source := g.FindPermanent(id)
		if source != nil && !source.PhasedOut && source.incarnationID == incarnation {
			identities = append(identities, capturedPermanent{id, incarnation})
		}
	}
	for _, modifier := range g.millModifiers {
		addRegistered(modifier.sourceID, modifier.sourceIncarnation, modifier.fn != nil)
	}
	for _, modifier := range g.lifeGainModifiers {
		addRegistered(modifier.sourceID, modifier.sourceIncarnation, modifier.fn != nil)
	}
	for _, source := range g.AllBattlefield() {
		if source.PhasedOut {
			continue
		}
		for _, ability := range source.RuntimeAbilities {
			switch modifier := UnwrapAbility(ability).(type) {
			case *staticMillReplacement:
				if modifier.factory != nil {
					identities = append(identities, capturedPermanent{source.ID(), source.incarnationID})
				}
			case *staticLifeGainReplacement:
				if modifier.factory != nil {
					identities = append(identities, capturedPermanent{source.ID(), source.incarnationID})
				}
			}
		}
	}
	return g.appendPermanentReferenceSources(nil, identities)
}

// ChooseDamageSourceFromWaitingModifierReferences uses explicit modifier sources.
// Other references held by callbacks need separate declarations.
func ChooseDamageSourceFromWaitingModifierReferences(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, waitingModifiers: true}
}
