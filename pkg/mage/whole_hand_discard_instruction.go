package mage

// WholeHandDiscardInstruction retains one whole-hand discard action.
// Supply exactly one singular player or player collection.
type WholeHandDiscardInstruction struct {
	Subject  PlayerRef
	Subjects PlayerSet
}

// Effect builds the shared discard action. Invalid subjects return nil.
func (instruction WholeHandDiscardInstruction) Effect() ResolvingEffect {
	singular, plural := !nilDefinition(instruction.Subject), !nilDefinition(instruction.Subjects)
	if singular == plural {
		return nil
	}
	if singular {
		return DiscardEntireHand(instruction.Subject)
	}
	return DiscardEach(instruction.Subjects)
}

// InspectWholeHandDiscardInstruction reads one native action without callbacks.
// Filtered discards, result bindings and composite parents use separate contracts.
func InspectWholeHandDiscardInstruction(effect ResolvingEffect) (WholeHandDiscardInstruction, bool) {
	var instruction WholeHandDiscardInstruction
	switch effect := effect.(type) {
	case *discardEntireHandEffect:
		if effect == nil || effect.invalid || effect.count != 0 || effect.result != nil {
			return instruction, false
		}
		instruction.Subject = effect.subject
	case *discardEachEffect:
		if effect == nil || effect.invalid || effect.resultCount != 0 || effect.result != nil {
			return instruction, false
		}
		instruction.Subjects = effect.subjects
	default:
		return instruction, false
	}
	return instruction, instruction.Effect() != nil
}

func (e *discardEntireHandEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.subject)
}
