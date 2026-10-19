package mage

// RequiredTapInstruction retains a mandatory tap and its separate branches.
// Execution uses the shared payment transaction and exact permanent visit.
type RequiredTapInstruction struct {
	Player       PlayerRef
	Subject      PermanentRef
	Paid, Unpaid ResolvingEffect
}

// Effect builds the native payment. Missing operands return nil.
func (instruction RequiredTapInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Player) || nilDefinition(instruction.Subject) || nilDefinition(instruction.Paid) && nilDefinition(instruction.Unpaid) {
		return nil
	}
	return RequiredTapOrElse(instruction.Player, instruction.Subject, instruction.Paid, instruction.Unpaid)
}

// InspectRequiredTapInstruction retains native operands without paying costs.
// The consumer must validate references and both branches before use.
func InspectRequiredTapInstruction(effect ResolvingEffect) (RequiredTapInstruction, bool) {
	payment, ok := effect.(*requiredTapPayment)
	if !ok || payment == nil {
		return RequiredTapInstruction{}, false
	}
	instruction := RequiredTapInstruction{Player: payment.player, Subject: payment.subject, Paid: payment.paid, Unpaid: payment.unpaid}
	return instruction, instruction.Effect() != nil
}

// Both branches validate before target gates. No payment outcome is predicted.
func inspectRequiredTapMana(instruction RequiredTapInstruction) ([]spellPermanentInstruction, bool) {
	if _, ok := instruction.Player.(controllerRef); !ok {
		return nil, false
	}
	if _, ok := instruction.Subject.(*PermanentTargetBinding); !ok {
		return nil, false
	}
	out := []spellPermanentInstruction{{subject: instruction.Subject, tapOperation: TapOperation}}
	for _, branch := range []ResolvingEffect{instruction.Paid, instruction.Unpaid} {
		if branch == nil {
			continue
		}
		productions, ok := InspectManaProductionInstructions(branch)
		if !ok {
			return nil, false
		}
		for _, production := range productions {
			plan, ok := inspectSpellManaProduction(production)
			if !ok {
				return nil, false
			}
			out = append(out, plan)
		}
	}
	return out, true
}

func inspectSpellManaProduction(instruction ManaProductionInstruction) (spellPermanentInstruction, bool) {
	plan := spellPermanentInstruction{manaSubject: instruction.Subject}
	if _, ok := InspectDamageRecipient(instruction.Subject); !ok {
		return plan, false
	}
	var ok bool
	plan.castingManaReads, plan.manaCharacteristics, ok = inspectManaAmount(instruction.Amount)
	return plan, ok
}

func inspectManaAmount(amount ResolutionValue) ([]*NumberBinding, []TargetCharacteristicValuePredicate, bool) {
	if reads, ok := castingManaReads(amount); ok {
		return reads, nil, true
	}
	if value, ok := amount.(permanentInformationValue); ok && value.mana {
		predicate, ok := InspectTargetCharacteristicValue(value)
		return nil, []TargetCharacteristicValuePredicate{predicate}, ok
	}
	if value, ok := amount.(arithmeticValue); ok && (value.operation == valueAdd || value.operation == valueMultiply) {
		left, leftValues, leftOK := inspectManaAmount(value.left)
		right, rightValues, rightOK := inspectManaAmount(value.right)
		if leftOK && rightOK {
			return append(left, right...), append(leftValues, rightValues...), true
		}
	}
	return nil, nil, false
}
