package mage

import (
	"slices"

	"github.com/google/uuid"
)

// StackTargetPermanentSources reads permanent targets of stack and resolving objects.
// Target legality cannot remove a saved reference before its stack object leaves.
// Other reference categories need separate contracts for a general source choice.
func (g *Game) StackTargetPermanentSources() []DamageSource {
	var sources []DamageSource
	for _, object := range g.stack.referenceObjects() {
		for _, id := range object.Targets {
			incarnation := object.targetIncarnations[id]
			if incarnation == uuid.Nil {
				continue
			}
			var source DamageSource
			p, lki := g.readPermanentIncarnationInformation(capturedPermanent{id, incarnation})
			if p != nil {
				source = damageSourceFromPermanent(p)
			} else if lki != nil {
				source = damageSourceFromLKI(lki)
			}
			if source.Valid() && !slices.ContainsFunc(sources, source.SameObject) {
				sources = append(sources, source)
			}
		}
	}
	return sources
}

// ChooseDamageSourceFromStackTargets chooses a permanent referred to by a target.
// This category alone cannot supply the complete CR 609.7a source list.
func ChooseDamageSourceFromStackTargets(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, stackTargets: true}
}
