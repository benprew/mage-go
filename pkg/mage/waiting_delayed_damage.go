package mage

import (
	"slices"

	"github.com/google/uuid"
)

func (g *Game) damageReferenceInformation(saved DamageSource) DamageSource {
	switch {
	case saved.Incarnation() != uuid.Nil && saved.StackID() == uuid.Nil:
		p, lki := g.readPermanentIncarnationInformation(capturedPermanent{saved.ID(), saved.Incarnation()})
		if p != nil {
			return damageSourceFromPermanent(p)
		}
		if lki != nil {
			return damageSourceFromLKI(lki)
		}
	case saved.StackID() != uuid.Nil && saved.Incarnation() == uuid.Nil:
		return g.spellReferenceInformation(saved)
	}
	return DamageSource{}
}

func (e *capturedDelayedBody) stackDamageBindings() []DamageSource {
	sources := make([]DamageSource, 0, len(e.damageSources))
	for binding, source := range e.damageSources {
		if binding != nil && source.Valid() && (source.Incarnation() != uuid.Nil) != (source.StackID() != uuid.Nil) {
			sources = append(sources, source)
		}
	}
	slices.SortFunc(sources, func(a, b DamageSource) int {
		for _, pair := range [][2]uuid.UUID{{a.ID(), b.ID()}, {a.Incarnation(), b.Incarnation()}, {a.StackID(), b.StackID()}} {
			if order := slices.Compare(pair[0][:], pair[1][:]); order != 0 {
				return order
			}
		}
		return 0
	})
	return sources
}

// WaitingDelayedDamageSources reads exact damage-source references in waiting frames.
// Current information must not change the saved event facts.
func (g *Game) WaitingDelayedDamageSources() []DamageSource {
	var sources []DamageSource
	appendEffects := func(effects []Effect) {
		for _, effect := range effects {
			if refs, ok := effect.(stackDamageBindingReferences); ok {
				for _, saved := range refs.stackDamageBindings() {
					source := g.damageReferenceInformation(saved)
					if source.SameObject(saved) && !slices.ContainsFunc(sources, source.SameObject) {
						sources = append(sources, cloneDamageSource(source))
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

// ChooseDamageSourceFromWaitingDelayedDamageBindings uses saved waiting source references.
// Other source categories need separate contracts for a general source choice.
func ChooseDamageSourceFromWaitingDelayedDamageBindings(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, waitingDelayedDamage: true}
}
