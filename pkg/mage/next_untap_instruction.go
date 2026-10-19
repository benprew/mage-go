package mage

// NextUntapInstruction retains a restriction for future controller untap steps.
// Exactly one subject field must be set. A collection supports one step.
type NextUntapInstruction struct {
	Subject  PermanentRef
	Subjects PermanentSet
	Count    int
}

// Effect builds the shared restriction. A malformed declaration returns nil.
func (instruction NextUntapInstruction) Effect() ResolvingEffect {
	singular, plural := !nilDefinition(instruction.Subject), !nilDefinition(instruction.Subjects)
	if singular == plural || instruction.Count < 0 {
		return nil
	}
	if singular {
		if instruction.Count == 1 {
			return SkipNextUntap(instruction.Subject)
		}
		return SkipNextUntaps(instruction.Subject, instruction.Count)
	}
	if instruction.Count != 1 {
		return nil
	}
	return SkipNextUntapEach(instruction.Subjects)
}

// InspectNextUntapInstruction reads a native declaration without selecting visits.
// Player-captured restrictions and other operations need separate contracts.
func InspectNextUntapInstruction(effect ResolvingEffect) (NextUntapInstruction, bool) {
	var instruction NextUntapInstruction
	switch effect := effect.(type) {
	case *skipNextUntapEffect:
		if effect == nil {
			return instruction, false
		}
		instruction = NextUntapInstruction{Subject: effect.subject, Count: 1}
	case *skipNextUntapsEffect:
		if effect == nil {
			return instruction, false
		}
		instruction = NextUntapInstruction{Subject: effect.subject, Count: effect.count}
	case *skipNextUntapEachEffect:
		if effect == nil {
			return instruction, false
		}
		instruction = NextUntapInstruction{Subjects: effect.subjects, Count: 1}
	default:
		return instruction, false
	}
	if instruction.Effect() == nil {
		return NextUntapInstruction{}, false
	}
	return instruction, true
}
