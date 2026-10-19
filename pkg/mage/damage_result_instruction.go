package mage

// DamageResultInstruction retains one damage action and its two saved results.
// Exactly one recipient form is required. Results use declaration identity.
type DamageResultInstruction struct {
	Source     DamageSourceRef
	Recipient  DamageRecipientRef
	Recipients DamageRecipientSet
	Amount     ResolutionValue
	Total      *NumberBinding
	Results    *DamageResultsBinding
}

// Effect builds shared damage mechanics. Missing or ambiguous operands return nil.
func (instruction DamageResultInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Source) || nilDefinition(instruction.Amount) || instruction.Total == nil || instruction.Total.name == "" || instruction.Results == nil || instruction.Results.name == "" {
		return nil
	}
	single, many := !nilDefinition(instruction.Recipient), !nilDefinition(instruction.Recipients)
	if single == many {
		return nil
	}
	if single {
		return DealDamageFrom(instruction.Source, instruction.Recipient, instruction.Amount, DamageAmountInto(instruction.Total), DamageOutcomesInto(instruction.Results))
	}
	return DealDamageEach(instruction.Source, instruction.Recipients, instruction.Amount, DamageTotalInto(instruction.Total), DamageResultsInto(instruction.Results))
}

// InspectDamageResultInstruction retains one complete native result producer.
// Inspection does not read saved results or execute any operand.
func InspectDamageResultInstruction(effect ResolvingEffect) (DamageResultInstruction, bool) {
	var instruction DamageResultInstruction
	switch effect := effect.(type) {
	case *damageFromEffect:
		if effect == nil || effect.withPrevention || effect.invalidOption || effect.amountOptionCount != 1 || effect.outcomesOptionCount != 1 || effect.playerOptionCount != 0 || effect.permanentOptionCount != 0 || effect.sourceOptionCount != 0 || effect.playerResult != nil || effect.permanentResult != nil || effect.sourceResult != nil {
			return instruction, false
		}
		instruction = DamageResultInstruction{Source: effect.source, Recipient: effect.recipient, Amount: effect.amount, Total: effect.result, Results: effect.outcomes}
	case *damageEachEffect:
		if effect == nil || effect.evenDivision || effect.invalidOption || effect.optionCount != 1 || effect.resultsCount != 1 {
			return instruction, false
		}
		instruction = DamageResultInstruction{Source: effect.source, Recipients: effect.recipients, Amount: effect.amount, Total: effect.total, Results: effect.results}
	default:
		return instruction, false
	}
	return instruction, instruction.Effect() != nil
}
