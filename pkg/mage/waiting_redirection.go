package mage

import "github.com/google/uuid"

// WaitingRedirectionPermanentSources reads exact references in typed redirection rules.
// Other replacement references require separate contracts.
func (g *Game) WaitingRedirectionPermanentSources() []DamageSource {
	return g.waitingReplacementPermanentSources(func(replacement ReplacementEffect) []capturedPermanent {
		if rule, ok := replacement.(*selectedSourceRedirectReplacement); ok {
			return []capturedPermanent{rule.subject, {rule.source.ID(), rule.source.Incarnation()}, {rule.continued.ID(), rule.continued.Incarnation()}}
		}
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

// WaitingRedirectionSpellSources reads exact spells in active selected-source rules.
// A later cast cannot replace the saved spell's information.
func (g *Game) WaitingRedirectionSpellSources() []DamageSource {
	return g.waitingReplacementSpellSources(func(replacement ReplacementEffect) []DamageSource {
		rule, ok := replacement.(*selectedSourceRedirectReplacement)
		if !ok || rule.source.StackID() == uuid.Nil || rule.source.Incarnation() != uuid.Nil {
			return nil
		}
		return []DamageSource{rule.source}
	})
}
