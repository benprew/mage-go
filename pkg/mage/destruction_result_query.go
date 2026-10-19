package mage

// DestructionResultInstruction retains one simultaneous graveyard-count producer.
// Result is a declaration identity, not a saved or predicted number.
type DestructionResultInstruction struct {
	Subjects         PermanentSet
	Result           *NumberBinding
	CannotRegenerate bool
}

// Effect builds the shared destruction action with one graveyard-count result.
// Missing subjects or an unnamed result returns nil.
func (instruction DestructionResultInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Subjects) || instruction.Result == nil || instruction.Result.name == "" {
		return nil
	}
	option := PermanentsPutIntoGraveyardThisWayInto(instruction.Result)
	if instruction.CannotRegenerate {
		return DestroyEachWithoutRegeneration(instruction.Subjects, option)
	}
	return DestroyEach(instruction.Subjects, option)
}

// InspectDestructionResult retains one native graveyard-count producer.
// Other result kinds, sequences and malformed options return false.
// Inspection does not select subjects or read resolution state.
func InspectDestructionResult(effect ResolvingEffect) (DestructionResultInstruction, bool) {
	e, ok := effect.(*destroyEachEffect)
	if !ok || e == nil || e.invalidOption || e.optionCount != 1 || e.creatureDeaths != nil || e.deathVisits != nil {
		return DestructionResultInstruction{}, false
	}
	instruction := DestructionResultInstruction{Subjects: e.subjects, Result: e.graveyardMoves, CannotRegenerate: e.cannotRegenerate}
	if instruction.Effect() == nil {
		return DestructionResultInstruction{}, false
	}
	return instruction, true
}

func destructionResultRead(value ResolutionValue) (*NumberBinding, bool) {
	saved, ok := value.(savedNumberValue)
	if !ok || saved.binding == nil || saved.binding.name == "" {
		return nil, false
	}
	return saved.binding, true
}
