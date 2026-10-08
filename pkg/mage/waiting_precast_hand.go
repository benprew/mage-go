package mage

import "github.com/google/uuid"

// WaitingPreventionHandSourcesFor reads exact visits in active prevention rules.
// A returned card cannot replace the saved visit.
func (g *Game) WaitingPreventionHandSourcesFor(viewer uuid.UUID) []DamageSource {
	return g.waitingReplacementZoneSources(func(replacement ReplacementEffect) []DamageSource {
		refs, ok := replacement.(waitingPreventionExactReferences)
		if !ok {
			return nil
		}
		return refs.waitingPreventionExactSources()
	}, DamageSource.HandVisit, func(saved DamageSource) DamageSource {
		return g.handReferenceInformationFor(viewer, saved)
	})
}

// WaitingRedirectionHandSourcesFor reads exact visits in active redirection rules.
// Consumption and cleanup end the reference lifetime.
func (g *Game) WaitingRedirectionHandSourcesFor(viewer uuid.UUID) []DamageSource {
	return g.waitingReplacementZoneSources(func(replacement ReplacementEffect) []DamageSource {
		refs, ok := replacement.(waitingRedirectionExactReferences)
		if !ok {
			return nil
		}
		return refs.waitingRedirectionExactSources()
	}, DamageSource.HandVisit, func(saved DamageSource) DamageSource {
		return g.handReferenceInformationFor(viewer, saved)
	})
}

// ChooseDamageSourceFromWaitingPrecastHandReferences uses saved hand visits.
// A hand source cannot follow a later spell or permanent.
func ChooseDamageSourceFromWaitingPrecastHandReferences(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, waitingPrecastHand: true}
}
