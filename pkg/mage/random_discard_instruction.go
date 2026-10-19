package mage

// RandomDiscardInstruction retains one singular native random discard action.
type RandomDiscardInstruction struct {
	Subject PlayerRef
	Amount  ResolutionValue
}

// Effect builds the native random discard action. Missing operands return nil.
func (instruction RandomDiscardInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Subject) || nilDefinition(instruction.Amount) {
		return nil
	}
	return DiscardAtRandom(instruction.Subject, instruction.Amount)
}

// InspectRandomDiscardInstructions reads complete native random discard trees without callbacks.
// Unknown children reject the complete tree.
func InspectRandomDiscardInstructions(effect ResolvingEffect) ([]RandomDiscardInstruction, bool) {
	var children []ResolvingEffect
	switch effect := effect.(type) {
	case *sequenceEffect:
		if effect == nil {
			return nil, false
		}
		children = effect.effects
	case *compositeEffect:
		if effect == nil {
			return nil, false
		}
		children = effect.effects
	case *randomDiscardSubjectEffect:
		if effect != nil {
			instruction := RandomDiscardInstruction{Subject: effect.subject, Amount: effect.amount}
			if instruction.Effect() != nil {
				return []RandomDiscardInstruction{instruction}, true
			}
		}
		return nil, false
	default:
		return nil, false
	}
	var out []RandomDiscardInstruction
	for _, child := range children {
		instructions, ok := InspectRandomDiscardInstructions(child)
		if !ok {
			return nil, false
		}
		out = append(out, instructions...)
	}
	return out, true
}
