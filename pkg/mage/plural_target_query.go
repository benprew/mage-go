package mage

import (
	"math"

	"github.com/google/uuid"
)

// PermanentTargetsPredicate retains one native plural permanent target declaration.
// It has no chosen members. The zero value cannot select targets.
type PermanentTargetsPredicate struct {
	target PermanentTargetPredicate
}

// InspectPermanentTargets accepts native fixed, exact-X or one-or-more targets.
// Target restrictions use the singular native contract without callbacks.
// One-or-more uses bounds 1..math.MaxInt. Other optional counts and wrappers reject.
func InspectPermanentTargets(binding *PermanentTargetsBinding) (PermanentTargetsPredicate, bool) {
	if binding == nil || binding.targetDeclaration == nil {
		return PermanentTargetsPredicate{}, false
	}
	var spec *PermanentTarget
	switch target := binding.spec.(type) {
	case *PermanentTarget:
		if target == nil || target.Min() < 0 || (target.Min() != target.Max() && (target.Min() != 1 || target.Max() != math.MaxInt)) {
			return PermanentTargetsPredicate{}, false
		}
		spec = target
	case *countBoundTarget:
		if target == nil || !target.usesX || target.upToX || target.oneToX || target.count != 0 {
			return PermanentTargetsPredicate{}, false
		}
		var ok bool
		spec, ok = target.inner.(*PermanentTarget)
		if !ok || spec == nil || spec.Min() != 1 || spec.Max() != 1 {
			return PermanentTargetsPredicate{}, false
		}
	default:
		return PermanentTargetsPredicate{}, false
	}
	target, ok := inspectPermanentTargetDeclaration(binding.targetDeclaration, spec)
	return PermanentTargetsPredicate{target: target}, ok
}

// Select checks every announced visit and current legality without callbacks.
// Missing declarations, visits, zones or count information return unknown.
// A zero-count declaration must still be present in the spell definition.
// Layers must be current. Results contain only legal exact visits.
func (p PermanentTargetsPredicate) Select(g *Game, obj *StackObject) ([]PermanentVisit, bool) {
	declaration := p.target.declaration
	if declaration == nil || g == nil || obj == nil || len(obj.TargetSpecs) != len(obj.Targets) || obj.XValue < 0 {
		return nil, false
	}
	minimum, maximum := TargetBounds(declaration, obj.XValue)
	if minimum < 0 || maximum < minimum {
		return nil, false
	}
	indices := make([]int, 0, len(obj.Targets))
	ids := make(map[uuid.UUID]bool)
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
	if len(indices) < minimum || len(indices) > maximum {
		return nil, false
	}
	if len(indices) == 0 {
		if nilDefinition(obj.Card) {
			return nil, false
		}
		found := false
		for _, spec := range obj.Card.CastTargets() {
			if spec == declaration {
				if found {
					return nil, false
				}
				found = true
			}
		}
		if !found {
			return nil, false
		}
	}
	var out []PermanentVisit
	for _, index := range indices {
		visits, known := p.target.selectIndex(g, obj, index)
		if !known {
			return nil, false
		}
		out = append(out, visits...)
	}
	return out, true
}
