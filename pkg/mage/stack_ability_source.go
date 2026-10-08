package mage

import (
	"slices"

	"github.com/google/uuid"
)

// StackAbilityPermanentSources reads original permanent sources of stack and resolving abilities.
// A returned card cannot replace the source saved by an ability. Other references
// and command-zone objects need separate contracts for a general source choice.
func (g *Game) StackAbilityPermanentSources() []DamageSource {
	var sources []DamageSource
	for _, object := range g.stack.referenceObjects() {
		if !object.IsAbility || object.sourceIncarnation == uuid.Nil {
			continue
		}
		var source DamageSource
		p, lki := g.readPermanentIncarnationInformation(capturedPermanent{object.SourceID, object.sourceIncarnation})
		if p != nil {
			source = damageSourceFromPermanent(p)
		} else if lki != nil {
			source = damageSourceFromLKI(lki)
		}
		if source.Valid() && !slices.ContainsFunc(sources, source.SameObject) {
			sources = append(sources, source)
		}
	}
	return sources
}

// ChooseDamageSourceFromStackAbilities chooses an original permanent ability source.
// Use this only for this source category. A general CR 609.7a choice needs
// other referred objects, live objects and face-up command-zone objects.
func ChooseDamageSourceFromStackAbilities(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, stackAbilities: true}
}
