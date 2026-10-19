package mage

// DirectReturnInstruction describes one return action without result bindings.
// Exactly one subject field must be set. Subjects remains one simultaneous group.
type DirectReturnInstruction struct {
	Subject  PermanentRef
	Subjects PermanentSet
}

// Effect builds the native action. A malformed declaration returns nil.
func (instruction DirectReturnInstruction) Effect() ResolvingEffect {
	singular, plural := !nilDefinition(instruction.Subject), !nilDefinition(instruction.Subjects)
	if singular == plural {
		return nil
	}
	if singular {
		return ReturnToHand(instruction.Subject)
	}
	return ReturnEachToHand(instruction.Subjects)
}

// InspectDirectReturn reads a complete tree of native return actions.
// Unknown children reject the whole tree. No operand or effect callbacks run.
func InspectDirectReturn(effect ResolvingEffect) ([]DirectReturnInstruction, bool) {
	instructions, ok := appendDirectReturn(nil, effect)
	if !ok {
		return nil, false
	}
	return instructions, true
}

func appendDirectReturn(out []DirectReturnInstruction, effect ResolvingEffect) ([]DirectReturnInstruction, bool) {
	switch effect := effect.(type) {
	case *returnToHandSubjectEffect:
		if effect == nil || nilDefinition(effect.subject) {
			return nil, false
		}
		return append(out, DirectReturnInstruction{Subject: effect.subject}), true
	case *returnEachToHandEffect:
		if effect == nil || nilDefinition(effect.subjects) {
			return nil, false
		}
		return append(out, DirectReturnInstruction{Subjects: effect.subjects}), true
	case *sequenceEffect:
		if effect == nil {
			return nil, false
		}
		return appendDirectReturnSequence(out, effect.effects)
	case *compositeEffect:
		if effect == nil {
			return nil, false
		}
		return appendDirectReturnSequence(out, effect.effects)
	default:
		return nil, false
	}
}

func appendDirectReturnSequence(out []DirectReturnInstruction, effects []ResolvingEffect) ([]DirectReturnInstruction, bool) {
	for _, effect := range effects {
		var ok bool
		out, ok = appendDirectReturn(out, effect)
		if !ok {
			return nil, false
		}
	}
	return out, true
}
