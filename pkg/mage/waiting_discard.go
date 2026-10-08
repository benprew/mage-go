package mage

// WaitingDiscardPermanentSources reads exact sources of active discard replacements.
// The replacement's live ability check also controls source availability.
func (g *Game) WaitingDiscardPermanentSources() []DamageSource {
	return g.waitingReplacementPermanentSources(func(replacement ReplacementEffect) []capturedPermanent {
		rule, ok := replacement.(*discardToLibraryReplacement)
		if !ok {
			return nil
		}
		return []capturedPermanent{{rule.sourceID, rule.incarnation}}
	})
}

// ChooseDamageSourceFromWaitingDiscardReferences uses explicit discard-rule sources.
// Other legal source categories need separate enumeration services.
func ChooseDamageSourceFromWaitingDiscardReferences(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, waitingDiscard: true}
}
