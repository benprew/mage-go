package mage

import "github.com/google/uuid"

// DamageTargetSelection keeps player identity separate from a permanent visit.
// A zero value reports an illegal target.
type DamageTargetSelection struct {
	Player    uuid.UUID
	Permanent PermanentVisit
}

// Empty reports whether the selection has no legal recipient.
func (selection DamageTargetSelection) Empty() bool {
	return selection.Player == uuid.Nil && selection.Permanent.ID == uuid.Nil
}

// DamageTargetPredicate retains one complete native damage target declaration.
// The declaration has no chosen state.
type DamageTargetPredicate struct {
	declaration      *targetDeclaration
	planeswalkerOnly bool
	opponentOnly     bool
}

// InspectDamageTarget reads singular any-target and player-or-planeswalker references.
// Chooser wrappers require separate query contracts. No callbacks run.
func InspectDamageTarget(reference *DamageTargetBinding) (DamageTargetPredicate, bool) {
	if reference == nil || reference.targetDeclaration == nil {
		return DamageTargetPredicate{}, false
	}
	result := DamageTargetPredicate{declaration: reference.targetDeclaration}
	switch spec := reference.spec.(type) {
	case *DamageAnyTarget:
		if spec == nil || spec.Min() != 1 || spec.Max() != 1 {
			return DamageTargetPredicate{}, false
		}
	case *PlayerOrPlaneswalkerTarget:
		if spec == nil || spec.Min() != 1 || spec.Max() != 1 {
			return DamageTargetPredicate{}, false
		}
		result.planeswalkerOnly = true
		result.opponentOnly = spec.opponentOnly
	default:
		return DamageTargetPredicate{}, false
	}
	return result, true
}

// Select reads current legality and exact announced identity without callbacks.
// Layers must be current. False reports an unknown query, not an illegal target.
// Permanent legality uses the shared native characteristic and source LKI checks.
func (predicate DamageTargetPredicate) Select(g *Game, obj *StackObject) (DamageTargetSelection, bool) {
	if predicate.declaration == nil || g == nil || obj == nil || g.GetPlayer(obj.Controller) == nil || len(obj.TargetSpecs) != len(obj.Targets) {
		return DamageTargetSelection{}, false
	}
	index := -1
	for i, spec := range obj.TargetSpecs {
		if declaration, ok := spec.(*targetDeclaration); ok && declaration == predicate.declaration {
			if index != -1 {
				return DamageTargetSelection{}, false
			}
			index = i
		}
	}
	if index < 0 {
		return DamageTargetSelection{}, false
	}
	id := obj.Targets[index]
	if player := g.GetPlayer(id); player != nil {
		if predicate.opponentOnly && id == obj.Controller {
			return DamageTargetSelection{}, true
		}
		return DamageTargetSelection{Player: id}, true
	}
	filter, _ := inspectContextualPermanentCollectionFilter(Or(IsCreature, IsPlaneswalker))
	if predicate.planeswalkerOnly {
		filter, _ = inspectContextualPermanentCollectionFilter(IsPlaneswalker)
	}
	permanent := PermanentTargetPredicate{declaration: predicate.declaration, filters: []*permanentCollectionFilterNode{filter}}
	visits, known := permanent.Select(g, obj)
	if !known || len(visits) == 0 {
		return DamageTargetSelection{}, known
	}
	return DamageTargetSelection{Permanent: visits[0]}, true
}
