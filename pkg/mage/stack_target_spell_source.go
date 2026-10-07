package mage

import "slices"

// StackTargetSpellSources reads spell targets referred to by objects on the stack.
// Departure information stays available until the last referring object leaves.
// This category alone cannot supply the complete CR 609.7a source list.
func (g *Game) StackTargetSpellSources() []DamageSource {
	var sources []DamageSource
	for _, object := range g.stack.objects {
		for _, id := range object.Targets {
			saved, ok := object.targetSpellSources[id]
			if !ok || !saved.Valid() || saved.StackID() != id {
				continue
			}
			source := g.CaptureSpellDamageSource(id)
			if !source.Valid() {
				source = cloneDamageSource(saved)
			}
			if !slices.ContainsFunc(sources, source.SameObject) {
				sources = append(sources, source)
			}
		}
	}
	return sources
}

// ChooseDamageSourceFromStackSpellTargets chooses a spell referred to by a target.
// General source choices also need other referred and command-zone objects.
func ChooseDamageSourceFromStackSpellTargets(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, stackSpellTargets: true}
}
