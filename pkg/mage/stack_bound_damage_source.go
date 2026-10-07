package mage

import (
	"maps"
	"slices"

	"github.com/google/uuid"
)

// StackBoundDamageSources reads source references saved by stack event captures.
// This category alone cannot supply the complete CR 609.7a source list.
func (g *Game) StackBoundDamageSources() []DamageSource {
	var sources []DamageSource
	for _, object := range g.stack.objects {
		for _, saved := range stackBoundDamageSources(object) {
			var source DamageSource
			switch {
			case saved.Incarnation() != uuid.Nil && saved.StackID() == uuid.Nil:
				p, lki := g.readPermanentIncarnationInformation(capturedPermanent{saved.ID(), saved.Incarnation()})
				if p != nil {
					source = damageSourceFromPermanent(p)
				} else if lki != nil {
					source = damageSourceFromLKI(lki)
				}
			case saved.StackID() != uuid.Nil && saved.Incarnation() == uuid.Nil:
				source = g.CaptureSpellDamageSource(saved.StackID())
				if !source.SameObject(saved) {
					source = object.boundSpellSources[saved.StackID()]
				}
				if !source.SameObject(saved) {
					source = saved
				}
			}
			if source.SameObject(saved) && !slices.ContainsFunc(sources, source.SameObject) {
				sources = append(sources, cloneDamageSource(source))
			}
		}
	}
	return sources
}

// ChooseDamageSourceFromStackDamageBindings chooses a source saved by a stack event.
// Other referred objects need separate contracts for a general source choice.
func ChooseDamageSourceFromStackDamageBindings(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, stackDamageBindings: true}
}

type stackDamageBindingReferences interface {
	stackDamageBindings() []DamageSource
}

func (e *eventDamageSourceCapture) stackDamageBindings() []DamageSource {
	if e.binding == nil || !e.source.Valid() {
		return nil
	}
	return []DamageSource{e.source}
}

func (e *sequenceEffect) stackDamageBindings() []DamageSource {
	var sources []DamageSource
	for _, effect := range e.effects {
		if references, ok := effect.(stackDamageBindingReferences); ok {
			sources = append(sources, references.stackDamageBindings()...)
		}
	}
	return sources
}

func stackBoundDamageSources(object *StackObject) []DamageSource {
	var sources []DamageSource
	for _, effect := range object.Effects {
		if references, ok := effect.(stackDamageBindingReferences); ok {
			sources = append(sources, references.stackDamageBindings()...)
		}
	}
	return sources
}

func (obj *StackObject) saveBoundSpellInformation(source DamageSource) {
	for _, saved := range stackBoundDamageSources(obj) {
		if saved.SameObject(source) {
			// Query information must not change the saved event information.
			obj.boundSpellSources = maps.Clone(obj.boundSpellSources)
			if obj.boundSpellSources == nil {
				obj.boundSpellSources = make(map[uuid.UUID]DamageSource)
			}
			obj.boundSpellSources[source.StackID()] = source
			return
		}
	}
}
