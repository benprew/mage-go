package mage

import (
	"slices"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// StackBoundSpellSources reads spell identities saved by stack event captures.
// This category alone cannot supply the complete CR 609.7a source list.
func (g *Game) StackBoundSpellSources() []DamageSource {
	var sources []DamageSource
	for _, object := range g.stack.objects {
		for _, saved := range stackBoundSpellSources(object) {
			source := g.CaptureSpellDamageSource(saved.StackID())
			if !source.SameObject(saved) {
				source = object.boundSpellSources[saved.StackID()]
			}
			if !source.SameObject(saved) {
				source = saved
			}
			if !slices.ContainsFunc(sources, source.SameObject) {
				sources = append(sources, cloneDamageSource(source))
			}
		}
	}
	return sources
}

// ChooseDamageSourceFromStackSpellBindings chooses a spell saved by a stack event.
// Other referred objects need separate contracts for a general source choice.
func ChooseDamageSourceFromStackSpellBindings(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, stackSpellBindings: true}
}

type stackSpellBindingReferences interface{ stackSpellBindings() []DamageSource }

func (e *eventSpellCapture) stackSpellBindings() []DamageSource {
	if e.binding == nil || !e.source.Valid() || e.id == uuid.Nil || e.source.StackID() != e.id || e.source.Incarnation() != uuid.Nil {
		return nil
	}
	return []DamageSource{e.source}
}

func (e *sequenceEffect) stackSpellBindings() []DamageSource {
	var sources []DamageSource
	for _, effect := range e.effects {
		if references, ok := effect.(stackSpellBindingReferences); ok {
			sources = append(sources, references.stackSpellBindings()...)
		}
	}
	return sources
}

func stackBoundSpellSources(object *StackObject) []DamageSource {
	var sources []DamageSource
	for _, effect := range object.Effects {
		if references, ok := effect.(stackSpellBindingReferences); ok {
			sources = append(sources, references.stackSpellBindings()...)
		}
	}
	return sources
}

func (p spellCastPattern) captureDetectedEvent(event *core.GameEvent, observer GameReader, _ uuid.UUID) []Effect {
	game, ok := observer.(*Game)
	if !ok || !p.captureSpell {
		return nil
	}
	return []Effect{&eventSpellCapture{binding: p.spell, id: event.TargetID, source: game.CaptureSpellDamageSource(event.TargetID)}}
}
