package mage

// WaitingEmptyLibraryPermanentSources reads exact sources of active draw rules.
// Queries do not run the replacement callback.
func (g *Game) WaitingEmptyLibraryPermanentSources() []DamageSource {
	return g.waitingReplacementPermanentSources(func(replacement ReplacementEffect) []capturedPermanent {
		rule, ok := replacement.(*emptyLibraryDrawReplacement)
		if !ok || rule.callback == nil {
			return nil
		}
		return []capturedPermanent{{rule.sourceID, rule.incarnation}}
	})
}

// ChooseDamageSourceFromWaitingEmptyLibraryReferences uses explicit draw-rule sources.
// Other references held by callbacks need separate declarations.
func ChooseDamageSourceFromWaitingEmptyLibraryReferences(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, waitingEmptyLibrary: true}
}
