package mage

// SingularDamageInstruction retains one native damage instruction.
// Result bindings and instruction-local prevention use separate declarations.
type SingularDamageInstruction struct {
	Source    DamageSourceRef
	Recipient DamageRecipientRef
	Amount    ResolutionValue
}

// Effect builds the native action. A missing operand returns nil.
func (instruction SingularDamageInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Source) || nilDefinition(instruction.Recipient) || nilDefinition(instruction.Amount) {
		return nil
	}
	return DealDamageFrom(instruction.Source, instruction.Recipient, instruction.Amount)
}

// InspectSingularDamageInstructions reads complete native damage trees without callbacks.
// Unknown children and result-bearing instructions reject the whole tree.
func InspectSingularDamageInstructions(effect ResolvingEffect) ([]SingularDamageInstruction, bool) {
	switch effect := effect.(type) {
	case *sequenceEffect:
		if effect != nil {
			return inspectSingularDamageSequence(effect.effects)
		}
	case *compositeEffect:
		if effect != nil {
			return inspectSingularDamageSequence(effect.effects)
		}
	case *damageFromEffect:
		if effect != nil && !effect.withPrevention && !effect.invalidOption && effect.amountOptionCount == 0 && effect.playerOptionCount == 0 && effect.permanentOptionCount == 0 && effect.sourceOptionCount == 0 && effect.outcomesOptionCount == 0 && effect.result == nil && effect.playerResult == nil && effect.permanentResult == nil && effect.sourceResult == nil && effect.outcomes == nil {
			instruction := SingularDamageInstruction{Source: effect.source, Recipient: effect.recipient, Amount: effect.amount}
			if instruction.Effect() != nil {
				return []SingularDamageInstruction{instruction}, true
			}
		}
	}
	return nil, false
}

func inspectSingularDamageSequence(effects []ResolvingEffect) ([]SingularDamageInstruction, bool) {
	var out []SingularDamageInstruction
	for _, effect := range effects {
		instructions, ok := InspectSingularDamageInstructions(effect)
		if !ok {
			return nil, false
		}
		out = append(out, instructions...)
	}
	return out, true
}

func pureSourceDamageOperands(source DamageSourceRef, amount ResolutionValue) bool {
	if _, ok := source.(sourceRef); !ok {
		return false
	}
	return pureFixedOrXAmount(amount)
}

func pureFixedOrXAmount(amount ResolutionValue) bool {
	switch value := amount.(type) {
	case fixedValue:
		return true
	case legacyResolutionValue:
		switch value.value.(type) {
		case fixedValue, xValue:
			return true
		}
	}
	return false
}
