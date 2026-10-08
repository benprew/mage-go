package mage

// WaitingPreventionExileSources reads exact visits in active prevention rules.
// A returned card cannot replace the saved visit.
func (g *Game) WaitingPreventionExileSources() []DamageSource {
	return g.waitingReplacementZoneSources(func(replacement ReplacementEffect) []DamageSource {
		refs, ok := replacement.(waitingPreventionExactReferences)
		if !ok {
			return nil
		}
		return refs.waitingPreventionExactSources()
	}, DamageSource.ExileVisit, g.exileReferenceInformation)
}

// WaitingRedirectionExileSources reads exact visits in active redirection rules.
// Consumption and cleanup end the reference lifetime.
func (g *Game) WaitingRedirectionExileSources() []DamageSource {
	return g.waitingReplacementZoneSources(func(replacement ReplacementEffect) []DamageSource {
		refs, ok := replacement.(waitingRedirectionExactReferences)
		if !ok {
			return nil
		}
		return refs.waitingRedirectionExactSources()
	}, DamageSource.ExileVisit, g.exileReferenceInformation)
}

// ChooseDamageSourceFromWaitingPrecastExileReferences uses saved exile visits.
// An exile source cannot follow a later spell or permanent.
func ChooseDamageSourceFromWaitingPrecastExileReferences(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, waitingPrecastExile: true}
}
