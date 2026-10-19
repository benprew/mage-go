package mage

// DirectDestructionInstruction describes one unconditional destruction action.
// Exactly one of Subject and Subjects is set. Subjects remains one simultaneous
// group. These references are declarations; inspection does not select objects.
type DirectDestructionInstruction struct {
	Subject          PermanentRef
	Subjects         PermanentSet
	CannotRegenerate bool
}

// Effect builds the native action for this declaration. A malformed declaration
// returns nil. Subject and Subjects must not both be set or both be absent.
func (instruction DirectDestructionInstruction) Effect() ResolvingEffect {
	singular, plural := !nilDefinition(instruction.Subject), !nilDefinition(instruction.Subjects)
	if singular == plural {
		return nil
	}
	if singular {
		if instruction.CannotRegenerate {
			return DestroyWithoutRegeneration(instruction.Subject)
		}
		return Destroy(instruction.Subject)
	}
	if instruction.CannotRegenerate {
		return DestroyEachWithoutRegeneration(instruction.Subjects)
	}
	return DestroyEach(instruction.Subjects)
}

// InspectDirectDestruction reads destruction actions without result bindings.
// Unsupported effects return nil, false, including unknown sequence children.
// The function does not execute effects, resolve references or call filters.
// The result describes instructions, not legal targets or destruction outcomes.
func InspectDirectDestruction(effect ResolvingEffect) ([]DirectDestructionInstruction, bool) {
	instructions, ok := appendDirectDestruction(nil, effect)
	if !ok {
		return nil, false
	}
	return instructions, true
}

func appendDirectDestruction(out []DirectDestructionInstruction, effect ResolvingEffect) ([]DirectDestructionInstruction, bool) {
	switch effect := effect.(type) {
	case *destroySubjectEffect:
		if effect == nil || nilDefinition(effect.subject) || effect.optionCount != 0 || effect.invalidOption {
			return nil, false
		}
		return append(out, DirectDestructionInstruction{Subject: effect.subject, CannotRegenerate: effect.cannotRegenerate}), true
	case *destroyEachEffect:
		if effect == nil || nilDefinition(effect.subjects) || effect.optionCount != 0 || effect.invalidOption {
			return nil, false
		}
		return append(out, DirectDestructionInstruction{Subjects: effect.subjects, CannotRegenerate: effect.cannotRegenerate}), true
	case *sequenceEffect:
		if effect == nil {
			return nil, false
		}
		return appendDirectDestructionSequence(out, effect.effects)
	case *compositeEffect:
		if effect == nil {
			return nil, false
		}
		return appendDirectDestructionSequence(out, effect.effects)
	default:
		return nil, false
	}
}

func appendDirectDestructionSequence(out []DirectDestructionInstruction, effects []ResolvingEffect) ([]DirectDestructionInstruction, bool) {
	for _, effect := range effects {
		var ok bool
		out, ok = appendDirectDestruction(out, effect)
		if !ok {
			return nil, false
		}
	}
	return out, true
}
