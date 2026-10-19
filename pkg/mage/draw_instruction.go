package mage

// DrawInstruction retains one singular native draw action.
// Result bindings use separate declarations.
type DrawInstruction struct {
	Subject PlayerRef
	Amount  ResolutionValue
}

// Effect builds the native draw action. Missing operands return nil.
func (instruction DrawInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Subject) || nilDefinition(instruction.Amount) {
		return nil
	}
	return DrawFor(instruction.Subject, instruction.Amount)
}

// InspectDrawInstructions reads complete native draw trees without callbacks.
// Unknown children and result bindings reject the complete tree.
func InspectDrawInstructions(effect ResolvingEffect) ([]DrawInstruction, bool) {
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
	case *drawSubjectEffect:
		if effect != nil && !effect.invalidOption && effect.optionCount == 0 && effect.cardCount == 0 && effect.setCount == 0 && effect.result == nil && effect.cardResult == nil && effect.cardsResult == nil {
			instruction := DrawInstruction{Subject: effect.subject, Amount: effect.amount}
			if instruction.Effect() != nil {
				return []DrawInstruction{instruction}, true
			}
		}
		return nil, false
	default:
		return nil, false
	}
	var out []DrawInstruction
	for _, child := range children {
		instructions, ok := InspectDrawInstructions(child)
		if !ok {
			return nil, false
		}
		out = append(out, instructions...)
	}
	return out, true
}
