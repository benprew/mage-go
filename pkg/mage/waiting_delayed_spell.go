package mage

import (
	"slices"

	"github.com/google/uuid"
)

func (g *Game) spellReferenceInformation(saved DamageSource) DamageSource {
	source := g.CaptureSpellDamageSource(saved.StackID())
	if !source.SameObject(saved) {
		source = g.stack.spellSources[saved.StackID()]
	}
	if !source.SameObject(saved) {
		source = saved
	}
	return cloneDamageSource(source)
}

func (g *Game) captureDelayedSpellSources() map[*SpellBinding]DamageSource {
	sources := make(map[*SpellBinding]DamageSource)
	for binding, id := range g.resolution.spellBindings {
		if binding == nil || id == uuid.Nil {
			continue
		}
		source := g.CaptureSpellDamageSource(id)
		if !source.Valid() {
			source = g.stack.spellSources[id]
		}
		if source.Valid() && source.StackID() == id && source.Incarnation() == uuid.Nil {
			sources[binding] = cloneDamageSource(source)
		}
	}
	return sources
}

func (e *capturedDelayedBody) stackSpellBindings() []DamageSource {
	var sources []DamageSource
	for binding, saved := range e.spellSources {
		if binding != nil && saved.Valid() && saved.StackID() != uuid.Nil && saved.Incarnation() == uuid.Nil && e.spells[binding] == saved.StackID() {
			sources = append(sources, saved)
		}
	}
	slices.SortFunc(sources, func(a, b DamageSource) int {
		ai, bi := a.StackID(), b.StackID()
		return slices.Compare(ai[:], bi[:])
	})
	return sources
}

// WaitingDelayedSpellSources reads exact spell references in waiting frames.
// Saved departure data cannot make a later cast replace the original spell.
func (g *Game) WaitingDelayedSpellSources() []DamageSource {
	var sources []DamageSource
	appendEffects := func(effects []Effect) {
		for _, effect := range effects {
			if refs, ok := effect.(stackSpellBindingReferences); ok {
				for _, saved := range refs.stackSpellBindings() {
					source := g.spellReferenceInformation(saved)
					if !slices.ContainsFunc(sources, source.SameObject) {
						sources = append(sources, source)
					}
				}
			}
		}
	}
	for _, delayed := range g.triggers.Delayed() {
		if delayed == nil {
			continue
		}
		appendEffects(delayed.Effects)
		for _, mode := range delayed.modes {
			appendEffects(mode.Effects)
		}
	}
	return sources
}

// ChooseDamageSourceFromWaitingDelayedSpellBindings uses saved waiting spell references.
// Other source categories need separate contracts for a general source choice.
func ChooseDamageSourceFromWaitingDelayedSpellBindings(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, waitingDelayedSpells: true}
}
