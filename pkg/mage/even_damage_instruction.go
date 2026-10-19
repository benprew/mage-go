package mage

// EvenDamageInstruction retains division at resolution with rounding down.
// Saved results and division at announcement require separate declarations.
type EvenDamageInstruction struct {
	Source     DamageSourceRef
	Recipients DamageRecipientSet
	Total      ResolutionValue
}

// Effect builds one simultaneous damage action. Missing operands return nil.
func (instruction EvenDamageInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Source) || nilDefinition(instruction.Recipients) || nilDefinition(instruction.Total) {
		return nil
	}
	return DealDamageEvenlyToRecipients(instruction.Source, instruction.Recipients, instruction.Total)
}

// InspectEvenDamageInstructions reads complete native trees without callbacks.
// Unknown children and result bindings reject the whole tree.
func InspectEvenDamageInstructions(effect ResolvingEffect) ([]EvenDamageInstruction, bool) {
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
	case *damageEachEffect:
		if effect != nil && effect.evenDivision && !effect.invalidOption && effect.optionCount == 0 && effect.resultsCount == 0 && effect.total == nil && effect.results == nil {
			instruction := EvenDamageInstruction{Source: effect.source, Recipients: effect.recipients, Total: effect.amount}
			if instruction.Effect() != nil {
				return []EvenDamageInstruction{instruction}, true
			}
		}
		return nil, false
	default:
		return nil, false
	}
	var out []EvenDamageInstruction
	for _, child := range children {
		instructions, ok := InspectEvenDamageInstructions(child)
		if !ok {
			return nil, false
		}
		out = append(out, instructions...)
	}
	return out, true
}
