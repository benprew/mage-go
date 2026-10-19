package mage

// PermanentExileInstruction describes one battlefield exile action without result bindings.
// Exactly one subject field must be set. Subjects remains one simultaneous group.
type PermanentExileInstruction struct {
	Subject  PermanentRef
	Subjects PermanentSet
}

// Effect builds the native action. A malformed declaration returns nil.
func (instruction PermanentExileInstruction) Effect() ResolvingEffect {
	singular, plural := !nilDefinition(instruction.Subject), !nilDefinition(instruction.Subjects)
	if singular == plural {
		return nil
	}
	if singular {
		return Exile(instruction.Subject)
	}
	return ExileEach(instruction.Subjects)
}

// InspectPermanentExileInstructions reads a complete tree of native battlefield exile actions.
// Unknown children reject the whole tree. No operand or effect callbacks run.
func InspectPermanentExileInstructions(effect ResolvingEffect) ([]PermanentExileInstruction, bool) {
	instructions, ok := appendPermanentExile(nil, effect)
	if !ok {
		return nil, false
	}
	return instructions, true
}

func appendPermanentExile(out []PermanentExileInstruction, effect ResolvingEffect) ([]PermanentExileInstruction, bool) {
	switch effect := effect.(type) {
	case *exileSubjectEffect:
		if effect == nil || nilDefinition(effect.subject) || effect.withResult || effect.result != nil || effect.optionCount != 0 || effect.invalidOption {
			return nil, false
		}
		return append(out, PermanentExileInstruction{Subject: effect.subject}), true
	case *exileEachEffect:
		if effect == nil || nilDefinition(effect.subjects) {
			return nil, false
		}
		return append(out, PermanentExileInstruction{Subjects: effect.subjects}), true
	case *sequenceEffect:
		if effect == nil {
			return nil, false
		}
		return appendPermanentExileSequence(out, effect.effects)
	case *compositeEffect:
		if effect == nil {
			return nil, false
		}
		return appendPermanentExileSequence(out, effect.effects)
	default:
		return nil, false
	}
}

func appendPermanentExileSequence(out []PermanentExileInstruction, effects []ResolvingEffect) ([]PermanentExileInstruction, bool) {
	for _, effect := range effects {
		var ok bool
		out, ok = appendPermanentExile(out, effect)
		if !ok {
			return nil, false
		}
	}
	return out, true
}
