package mage

// LifeInstruction retains one singular native life action.
// Result bindings use separate declarations.
type LifeInstruction struct {
	Subject PlayerRef
	Amount  ResolutionValue
	Gain    bool
}

// Effect builds the native life action. Missing operands return nil.
func (instruction LifeInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Subject) || nilDefinition(instruction.Amount) {
		return nil
	}
	if instruction.Gain {
		return GainLifeFor(instruction.Subject, instruction.Amount)
	}
	return LoseLifeFor(instruction.Subject, instruction.Amount)
}

// InspectLifeInstructions reads complete native life trees without callbacks.
// Unknown children and result bindings reject the complete tree.
func InspectLifeInstructions(effect ResolvingEffect) ([]LifeInstruction, bool) {
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
	case *lifeAction:
		if effect != nil && !effect.gainResult && effect.lifeGainResultState == (lifeGainResultState{}) && effect.lifeLossResultState == (lifeLossResultState{}) {
			instruction := LifeInstruction{Subject: effect.subject, Amount: effect.amount, Gain: effect.gain}
			if instruction.Effect() != nil {
				return []LifeInstruction{instruction}, true
			}
		}
		return nil, false
	default:
		return nil, false
	}
	var out []LifeInstruction
	for _, child := range children {
		instructions, ok := InspectLifeInstructions(child)
		if !ok {
			return nil, false
		}
		out = append(out, instructions...)
	}
	return out, true
}
