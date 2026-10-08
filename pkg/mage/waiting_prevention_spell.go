package mage

import (
	"slices"

	"github.com/google/uuid"
)

type waitingPreventionSpellReferences interface {
	waitingPreventionSpells() []DamageSource
}

func (r *sourcePreventionReplacement) waitingPreventionSpells() []DamageSource {
	source := r.exactSource
	if !source.Valid() || source.StackID() == uuid.Nil || source.Incarnation() != uuid.Nil {
		return nil
	}
	return []DamageSource{source}
}

// WaitingPreventionSpellSources reads exact spell references in active shields.
// A later cast cannot replace the saved spell's information.
func (g *Game) WaitingPreventionSpellSources() []DamageSource {
	var sources []DamageSource
	for _, list := range [2][]ReplacementEffect{g.effects.replacements, g.effects.cycleReplacements} {
		for _, replacement := range list {
			if !replacement.IsActive(g) {
				continue
			}
			refs, ok := replacement.(waitingPreventionSpellReferences)
			if !ok {
				continue
			}
			for _, saved := range refs.waitingPreventionSpells() {
				source := g.spellReferenceInformation(saved)
				if source.SameObject(saved) && !slices.ContainsFunc(sources, source.SameObject) {
					sources = append(sources, cloneDamageSource(source))
				}
			}
		}
	}
	return sources
}

// ChooseDamageSourceFromWaitingPreventionSpellReferences uses exact shield references.
// Other source categories require separate enumeration services.
func ChooseDamageSourceFromWaitingPreventionSpellReferences(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, waitingPreventionSpells: true}
}
