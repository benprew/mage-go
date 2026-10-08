package mage

// WaitingRegenerationPermanentSources reads exact subjects of active regeneration shields.
// Departure and phasing do not remove a waiting shield's saved reference.
func (g *Game) WaitingRegenerationPermanentSources() []DamageSource {
	return g.waitingReplacementPermanentSources(func(replacement ReplacementEffect) []capturedPermanent {
		shield, ok := replacement.(*regenerationReplacement)
		if !ok {
			return nil
		}
		return []capturedPermanent{{shield.permanentID, shield.incarnation}}
	})
}

// ChooseDamageSourceFromWaitingRegenerationReferences uses saved regeneration subjects.
// Other legal source categories require separate enumeration services.
func ChooseDamageSourceFromWaitingRegenerationReferences(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, waitingRegeneration: true}
}
