package mage

import (
	"slices"

	"github.com/google/uuid"
)

// DamageSourcesFor reads the CR 609.7a source categories for a choosing player.
// Hand references use that player's inspection rights. Unreferred private cards
// are excluded. Exact identities keep returned objects and separate casts apart.
// Each result has separate card data. The query does not make a choice.
func (g *Game) DamageSourcesFor(playerID uuid.UUID) []DamageSource {
	candidates := (&damageSourceChoice{allSources: true}).candidates(g, playerID)
	candidates = append(candidates, g.WaitingDamageExpansionSources()...)
	for _, object := range g.stack.referenceObjects() {
		if object.IsAbility && object.abilitySourceVisit != uuid.Nil {
			candidates = append(candidates, g.graveyardVisitInformation(object.SourceID, object.abilitySourceVisit))
		}
	}
	sources := make([]DamageSource, 0, len(candidates))
	for _, source := range candidates {
		if source.HandVisit() != uuid.Nil {
			source = g.handReferenceInformationFor(playerID, source)
		}
		if !source.hasExactIdentity() || slices.ContainsFunc(sources, source.SameObject) {
			continue
		}
		sources = append(sources, cloneDamageSource(source))
	}
	return sources
}

// ChooseDamageSource saves one non-targeted CR 609.7a choice at resolution.
// The filter reads source information available to the choosing player.
// Replacement effects must check source properties again before damage.
func ChooseDamageSource(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, allSources: true}
}
