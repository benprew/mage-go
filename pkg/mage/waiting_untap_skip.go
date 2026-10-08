package mage

// WaitingUntapSkipPermanentSources reads exact sources of active step replacements.
// The replacement's live ability check also controls source availability.
func (g *Game) WaitingUntapSkipPermanentSources() []DamageSource {
	return g.waitingReplacementPermanentSources(func(replacement ReplacementEffect) []capturedPermanent {
		rule, ok := replacement.(*untapStepSkipReplacement)
		if !ok {
			return nil
		}
		return []capturedPermanent{{rule.sourceID, rule.incarnation}}
	})
}

// ChooseDamageSourceFromWaitingUntapSkipReferences uses explicit step-rule sources.
// Other legal source categories need separate enumeration services.
func ChooseDamageSourceFromWaitingUntapSkipReferences(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, waitingUntapSkip: true}
}
