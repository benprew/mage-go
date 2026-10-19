package mage

import "github.com/google/uuid"

// PlayerTargetPredicate retains one complete singular player declaration.
// The declaration stores no chosen player.
type PlayerTargetPredicate struct {
	declaration      *targetDeclaration
	opponent         bool
	attackedThisTurn bool
}

// InspectPlayerTarget reads native player declarations, including attack history.
// Unknown wrappers need separate inspection contracts.
func InspectPlayerTarget(reference PlayerRef) (PlayerTargetPredicate, bool) {
	binding, ok := reference.(*PlayerTargetBinding)
	if !ok || binding == nil || binding.targetDeclaration == nil {
		return PlayerTargetPredicate{}, false
	}
	result := PlayerTargetPredicate{declaration: binding.targetDeclaration}
	switch spec := binding.spec.(type) {
	case *PlayerTarget:
		if spec == nil || spec.Min() != 1 || spec.Max() != 1 {
			return PlayerTargetPredicate{}, false
		}
		result.attackedThisTurn = spec.attackedThisTurn
	case *OpponentTarget:
		if spec == nil || spec.Min() != 1 || spec.Max() != 1 {
			return PlayerTargetPredicate{}, false
		}
		result.opponent = true
		result.attackedThisTurn = spec.attackedThisTurn
	default:
		return PlayerTargetPredicate{}, false
	}
	return result, true
}

// Select reads the announced player and current native legality without callbacks.
// A nil identity with a true known result means that the target is illegal.
// Missing or repeated declarations return unknown. No player LKI is used.
func (p PlayerTargetPredicate) Select(g *Game, obj *StackObject) (uuid.UUID, bool) {
	if p.declaration == nil || g == nil || obj == nil || g.GetPlayer(obj.Controller) == nil || len(obj.TargetSpecs) != len(obj.Targets) {
		return uuid.Nil, false
	}
	index := -1
	for i, spec := range obj.TargetSpecs {
		if declaration, ok := spec.(*targetDeclaration); ok && declaration == p.declaration {
			if index != -1 {
				return uuid.Nil, false
			}
			index = i
		}
	}
	if index < 0 {
		return uuid.Nil, false
	}
	id := obj.Targets[index]
	if g.GetPlayer(id) == nil || p.opponent && id == obj.Controller || p.attackedThisTurn && !g.PlayerAttackedThisTurn(id) {
		return uuid.Nil, true
	}
	return id, true
}
