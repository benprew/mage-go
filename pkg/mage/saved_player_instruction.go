package mage

import "slices"

// SavedPlayerInstruction retains a player snapshot and its following body.
// Execution uses the shared resolution frame and delayed frame capture.
type SavedPlayerInstruction struct {
	Subject PlayerRef
	Binding *PlayerBinding
	Body    ResolvingEffect
}

// Effect builds the ordered snapshot and body. Missing operands return nil.
func (instruction SavedPlayerInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Subject) || instruction.Binding == nil || instruction.Binding.name == "" || nilDefinition(instruction.Body) {
		return nil
	}
	return Sequence(SnapshotPlayer(instruction.Subject, instruction.Binding), instruction.Body)
}

// InspectSavedPlayerInstruction reads the declaration without reading a player.
// Consumers must check the body and the exact binding of each saved read.
func InspectSavedPlayerInstruction(effect ResolvingEffect) (SavedPlayerInstruction, bool) {
	e, ok := effect.(*sequenceEffect)
	if !ok || e == nil || len(e.effects) != 2 {
		return SavedPlayerInstruction{}, false
	}
	snapshot, ok := e.effects[0].(*playerSnapshot)
	if !ok || snapshot == nil {
		return SavedPlayerInstruction{}, false
	}
	instruction := SavedPlayerInstruction{Subject: snapshot.subject, Binding: snapshot.binding, Body: e.effects[1]}
	return instruction, instruction.Effect() != nil
}

func inspectSavedPlayerDelayedDestruction(instruction SavedPlayerInstruction) bool {
	switch instruction.Subject.(type) {
	case controllerRef, activePlayerRef:
	default:
		return false
	}
	return inspectSavedPlayerSchedules(instruction.Body, instruction.Binding)
}

func inspectSavedPlayerSchedules(effect ResolvingEffect, binding *PlayerBinding) bool {
	if sequence, ok := effect.(*sequenceEffect); ok {
		if sequence == nil || len(sequence.effects) == 0 {
			return false
		}
		for _, child := range sequence.effects {
			if !inspectSavedPlayerSchedules(child, binding) {
				return false
			}
		}
		return true
	}
	delay, ok := InspectNextEndStepInstruction(effect)
	if !ok {
		return false
	}
	instructions, ok := InspectDirectDestruction(delay.Body)
	if !ok || len(instructions) == 0 {
		return false
	}
	for _, destruction := range instructions {
		if destruction.Subject != nil {
			return false
		}
		root, ok := inspectContextualCollectionWithMember(destruction.Subjects, binding)
		if !ok || collectionReadsTargets(root) {
			return false
		}
	}
	return true
}

func collectionReadsTargets(node *contextualCollectionNode) bool {
	if node.playerTarget != nil || node.permanentTarget != nil || node.blockedByTarget != nil {
		return true
	}
	if node.hosts != nil && collectionReadsTargets(node.hosts) {
		return true
	}
	return slices.ContainsFunc(node.children, collectionReadsTargets)
}
