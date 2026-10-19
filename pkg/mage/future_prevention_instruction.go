package mage

// FuturePreventionInstruction retains one finite recipient shield installation.
// The shield lasts until cleanup or until its amount is used.
type FuturePreventionInstruction struct {
	Recipient DamageRecipientRef
	Amount    ResolutionValue
}

// Effect builds the native installation. Missing operands return nil.
func (instruction FuturePreventionInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Recipient) || nilDefinition(instruction.Amount) {
		return nil
	}
	return PreventNextDamage(instruction.Recipient, instruction.Amount)
}

// InspectFuturePreventionInstructions reads complete native installation trees.
// It does not resolve operands or install shields. Unknown children reject the tree.
func InspectFuturePreventionInstructions(effect ResolvingEffect) ([]FuturePreventionInstruction, bool) {
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
	case *preventionInstallation:
		if effect != nil {
			instruction := FuturePreventionInstruction{Recipient: effect.recipient, Amount: effect.amount}
			if instruction.Effect() != nil {
				return []FuturePreventionInstruction{instruction}, true
			}
		}
		return nil, false
	default:
		return nil, false
	}
	var out []FuturePreventionInstruction
	for _, child := range children {
		instructions, ok := InspectFuturePreventionInstructions(child)
		if !ok {
			return nil, false
		}
		out = append(out, instructions...)
	}
	return out, true
}
