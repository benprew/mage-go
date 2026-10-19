package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// ObjectTargetSelection keeps stack identity separate from a permanent visit.
// A zero value means that the target is illegal.
type ObjectTargetSelection struct {
	Spell     uuid.UUID
	Permanent PermanentVisit
}

// Empty reports whether the target has no legal object.
func (selection ObjectTargetSelection) Empty() bool {
	return selection.Spell == uuid.Nil && selection.Permanent.ID == uuid.Nil
}

// ObjectTargetPredicate retains one native spell-or-permanent declaration.
type ObjectTargetPredicate struct{ declaration *targetDeclaration }

// InspectObjectTarget reads one complete union declaration without callbacks.
func InspectObjectTarget(binding *SpellOrPermanentTargetBinding) (ObjectTargetPredicate, bool) {
	if binding == nil || binding.targetDeclaration == nil || binding.name == "" {
		return ObjectTargetPredicate{}, false
	}
	spec, ok := binding.spec.(*typedSpellOrPermanentTarget)
	if !ok || spec == nil || spec.Min() != 1 || spec.Max() != 1 {
		return ObjectTargetPredicate{}, false
	}
	return ObjectTargetPredicate{declaration: binding.targetDeclaration}, true
}

// Select reads the announced category, exact identity and current legality.
// Missing snapshots or repeated declarations are unknown. No LKI fallback is used.
func (predicate ObjectTargetPredicate) Select(g *Game, obj *StackObject) (ObjectTargetSelection, bool) {
	if g == nil || obj == nil || g.GetPlayer(obj.Controller) == nil || len(obj.TargetSpecs) != len(obj.Targets) {
		return ObjectTargetSelection{}, false
	}
	if _, ok := InspectObjectTarget(&SpellOrPermanentTargetBinding{predicate.declaration}); !ok {
		return ObjectTargetSelection{}, false
	}
	index := -1
	for i, spec := range obj.TargetSpecs {
		if spec == predicate.declaration {
			if index != -1 {
				return ObjectTargetSelection{}, false
			}
			index = i
		}
	}
	if index < 0 {
		return ObjectTargetSelection{}, false
	}
	id := obj.Targets[index]
	if id == uuid.Nil {
		return ObjectTargetSelection{}, false
	}
	zone, ok := obj.TargetZones[id]
	if !ok {
		return ObjectTargetSelection{}, false
	}
	switch zone {
	case core.ZoneBattlefield:
		permanent := PermanentTargetPredicate{declaration: predicate.declaration}
		visits, known := permanent.Select(g, obj)
		if !known || len(visits) == 0 {
			return ObjectTargetSelection{}, known
		}
		return ObjectTargetSelection{Permanent: visits[0]}, true
	case core.ZoneStack:
		if obj.targetIncarnations[id] != uuid.Nil {
			return ObjectTargetSelection{}, false
		}
		spell := g.stack.FindByID(id)
		if spell == nil || spell.leftStack || spell.IsAbility || nilDefinition(spell.Card) {
			return ObjectTargetSelection{}, true
		}
		return ObjectTargetSelection{Spell: id}, true
	default:
		return ObjectTargetSelection{}, false
	}
}
