package mage

// NextCombatThisTurnInstruction retains one end-of-combat schedule before cleanup.
// Execution uses the shared delayed frame and resolving controller.
type NextCombatThisTurnInstruction struct{ Body ResolvingEffect }

// Effect builds the native schedule. A missing body returns nil.
func (instruction NextCombatThisTurnInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Body) {
		return nil
	}
	return AtNextThisTurn(EachEndOfCombat, instruction.Body)
}

// InspectNextCombatThisTurnInstruction reads the body without creating a trigger.
// Consumers must validate every child before they use the declaration.
func InspectNextCombatThisTurnInstruction(effect ResolvingEffect) (NextCombatThisTurnInstruction, bool) {
	e, ok := effect.(*nextStepEffect)
	if !ok || e == nil || e.repeats || !e.expiresAtCleanup || e.condition != nil || len(e.modes) != 0 || len(e.targets) != 0 || nilDefinition(e.body) {
		return NextCombatThisTurnInstruction{}, false
	}
	pattern, ok := e.pattern.(stepPattern)
	if !ok || pattern != EachEndOfCombat {
		return NextCombatThisTurnInstruction{}, false
	}
	return NextCombatThisTurnInstruction{Body: e.body}, true
}

func inspectCombatThisTurnDestruction(effect ResolvingEffect) ([]spellPermanentInstruction, bool) {
	instructions, ok := InspectDirectDestruction(effect)
	if !ok || len(instructions) == 0 {
		return nil, false
	}
	out := make([]spellPermanentInstruction, 0, len(instructions))
	for _, instruction := range instructions {
		if instruction.Subject != nil {
			target, ok := instruction.Subject.(*PermanentTargetBinding)
			if !ok || target == nil {
				return nil, false
			}
			if _, ok := InspectPermanentTarget(target); !ok {
				return nil, false
			}
		} else if _, ok := InspectContextualPermanentCollection(instruction.Subjects); !ok {
			return nil, false
		}
		out = append(out, spellPermanentInstruction{deferred: true, subject: instruction.Subject, subjects: instruction.Subjects})
	}
	return out, true
}
