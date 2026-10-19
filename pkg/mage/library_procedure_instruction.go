package mage

// LibraryProcedureInstruction retains a fixed look or an optional shuffle.
// Chooser views and orders the cards or decides whether Library shuffles.
type LibraryProcedureInstruction struct {
	Chooser, Library PlayerRef
	Count            int
	Reorder          bool
	OptionalShuffle  bool
}

// Effect builds the procedure through the shared private library services.
// Missing operands and incompatible operations return nil.
func (instruction LibraryProcedureInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Chooser) || nilDefinition(instruction.Library) || instruction.Count < 0 {
		return nil
	}
	if instruction.OptionalShuffle {
		if instruction.Count != 0 || instruction.Reorder {
			return nil
		}
		return MayShuffleLibrary(instruction.Chooser, instruction.Library)
	}
	return LookAtLibraryTop(instruction.Chooser, instruction.Library, instruction.Count, instruction.Reorder)
}

// InspectLibraryProcedureInstruction reads a complete native library operation.
// Dynamic counts and composite trees need separate inspection contracts.
func InspectLibraryProcedureInstruction(effect ResolvingEffect) (LibraryProcedureInstruction, bool) {
	var instruction LibraryProcedureInstruction
	switch effect := effect.(type) {
	case *libraryInspectionEffect:
		if effect == nil {
			return instruction, false
		}
		count, ok := effect.count.(fixedValue)
		if !ok {
			return instruction, false
		}
		instruction = LibraryProcedureInstruction{Chooser: effect.viewer, Library: effect.library, Count: count.n, Reorder: effect.reorder}
	case *mayShuffleLibraryEffect:
		if effect == nil {
			return instruction, false
		}
		instruction = LibraryProcedureInstruction{Chooser: effect.chooser, Library: effect.library, OptionalShuffle: true}
	default:
		return instruction, false
	}
	if instruction.Effect() == nil {
		return LibraryProcedureInstruction{}, false
	}
	return instruction, true
}
