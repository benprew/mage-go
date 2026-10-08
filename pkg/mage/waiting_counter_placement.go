package mage

// WaitingCounterPlacementPermanentSources reads explicit source exclusions in entry rules.
// A general filter does not refer to each permanent that can match it.
func (g *Game) WaitingCounterPlacementPermanentSources() []DamageSource {
	return g.waitingReplacementPermanentSources(func(replacement ReplacementEffect) []capturedPermanent {
		rule, ok := replacement.(*etbAdditionalCountersReplacement)
		if !ok || !rule.excludeSelf {
			return nil
		}
		return []capturedPermanent{{rule.sourceID, rule.incarnation}}
	})
}

// ChooseDamageSourceFromWaitingCounterPlacementReferences uses explicit entry-rule references.
// Other legal source categories need separate enumeration services.
func ChooseDamageSourceFromWaitingCounterPlacementReferences(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, waitingCounterPlacement: true}
}
