package mage

import (
	"github.com/google/uuid"
)

// DamageTargetsPredicate retains one native fixed or optional damage target group.
// It has no chosen members. The zero value cannot select targets.
type DamageTargetsPredicate struct{ declaration *targetDeclaration }

// InspectDamageTargets reads a complete native group without callbacks.
// X bounds and chooser wrappers require separate contracts.
func InspectDamageTargets(binding *DamageTargetsBinding) (DamageTargetsPredicate, bool) {
	if binding == nil || binding.targetDeclaration == nil {
		return DamageTargetsPredicate{}, false
	}
	spec, ok := binding.spec.(*DamageAnyTarget)
	if !ok || spec == nil || spec.Min() < 0 || spec.Max() < spec.Min() {
		return DamageTargetsPredicate{}, false
	}
	return DamageTargetsPredicate{declaration: binding.targetDeclaration}, true
}

// Select reads current legality and exact announced visits without callbacks.
// Layers must be current. A missing frame or identity returns unknown.
func (predicate DamageTargetsPredicate) Select(g *Game, obj *StackObject) ([]DamageTargetSelection, bool) {
	declaration := predicate.declaration
	if declaration == nil || g == nil || obj == nil || g.GetPlayer(obj.Controller) == nil || len(obj.TargetSpecs) != len(obj.Targets) {
		return nil, false
	}
	ids := make(map[uuid.UUID]bool)
	var indices []int
	for i, spec := range obj.TargetSpecs {
		if spec != declaration {
			continue
		}
		id := obj.Targets[i]
		if id == uuid.Nil || ids[id] {
			return nil, false
		}
		ids[id] = true
		indices = append(indices, i)
	}
	if len(indices) < declaration.Min() || len(indices) > declaration.Max() {
		return nil, false
	}
	if len(indices) == 0 {
		if nilDefinition(obj.Card) {
			return nil, false
		}
		count := 0
		for _, target := range obj.Card.CastTargets() {
			if target == declaration {
				count++
			}
		}
		if count != 1 {
			return nil, false
		}
	}
	filter, _ := inspectContextualPermanentCollectionFilter(Or(IsCreature, IsPlaneswalker))
	permanent := PermanentTargetPredicate{declaration: declaration, filters: []*permanentCollectionFilterNode{filter}}
	var out []DamageTargetSelection
	for _, index := range indices {
		id := obj.Targets[index]
		if g.GetPlayer(id) != nil {
			out = append(out, DamageTargetSelection{Player: id})
			continue
		}
		visits, known := permanent.selectIndex(g, obj, index)
		if !known {
			return nil, false
		}
		for _, visit := range visits {
			out = append(out, DamageTargetSelection{Permanent: visit})
		}
	}
	return out, true
}
