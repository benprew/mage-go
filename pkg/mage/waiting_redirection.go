package mage

// WaitingRedirectionPermanentSources reads exact references in typed redirection rules.
// Other replacement references require separate contracts.
func (g *Game) WaitingRedirectionPermanentSources() []DamageSource {
	return g.waitingReplacementPermanentSources(func(replacement ReplacementEffect) []capturedPermanent {
		if rule, ok := replacement.(*permanentDamageRedirectReplacement); ok {
			return []capturedPermanent{rule.from, rule.to}
		}
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

// ChooseDamageSourceFromWaitingRedirectionReferences uses exact waiting references.
// The caller needs separate services for other legal source categories.
func ChooseDamageSourceFromWaitingRedirectionReferences(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, waitingRedirection: true}
}
