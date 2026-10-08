package mage

// WaitingRedirectionPermanentSources reads live typed static redirection destinations.
// Other replacement references require separate contracts.
func (g *Game) WaitingRedirectionPermanentSources() []DamageSource {
	return g.waitingReplacementPermanentSources(func(replacement ReplacementEffect) []capturedPermanent {
		rule, ok := replacement.(*staticDamageRedirectReplacement)
		if !ok {
			return nil
		}
		_, subject := rule.context(g)
		if subject == nil {
			return nil
		}
		return []capturedPermanent{{subject.ID(), subject.incarnationID}}
	})
}

// ChooseDamageSourceFromWaitingRedirectionReferences uses live redirection destinations.
// The caller needs separate services for other legal source categories.
func ChooseDamageSourceFromWaitingRedirectionReferences(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, waitingRedirection: true}
}
