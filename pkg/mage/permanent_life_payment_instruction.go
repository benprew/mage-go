package mage

// PermanentLifePaymentInstruction retains optional life offers and their body.
// The unpaid binding identifies exact visits in the shared resolution frame.
type PermanentLifePaymentInstruction struct {
	Subjects PermanentSet
	Amount   int
	Unpaid   *PermanentSetBinding
	Body     ResolvingEffect
}

// Effect builds the shared payment and ordered continuation.
// Missing operands, unnamed bindings and negative amounts return nil.
func (instruction PermanentLifePaymentInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Subjects) || instruction.Amount < 0 || instruction.Unpaid == nil || instruction.Unpaid.name == "" || nilDefinition(instruction.Body) {
		return nil
	}
	return Sequence(OfferCostForEachPermanent(instruction.Subjects, EachPlayer, Costs(LifePayCost(instruction.Amount)), instruction.Unpaid), instruction.Body)
}

// InspectPermanentLifePaymentInstruction reads declarations without making offers.
// Consumers must validate the complete body and each saved binding before use.
func InspectPermanentLifePaymentInstruction(effect ResolvingEffect) (PermanentLifePaymentInstruction, bool) {
	sequence, ok := effect.(*sequenceEffect)
	if !ok || sequence == nil || len(sequence.effects) != 2 {
		return PermanentLifePaymentInstruction{}, false
	}
	payment, ok := sequence.effects[0].(*permanentPayment)
	if !ok || payment == nil || len(payment.costs.costs) != 1 {
		return PermanentLifePaymentInstruction{}, false
	}
	if _, ok := payment.players.(allPlayersRef); !ok {
		return PermanentLifePaymentInstruction{}, false
	}
	cost, ok := payment.costs.costs[0].(*lifePayCost)
	if !ok || cost == nil || cost.capture || cost.paid != nil {
		return PermanentLifePaymentInstruction{}, false
	}
	instruction := PermanentLifePaymentInstruction{Subjects: payment.subjects, Amount: cost.amount, Unpaid: payment.unpaid, Body: sequence.effects[1]}
	return instruction, instruction.Effect() != nil
}

func inspectOptionalUnpaidDestruction(instruction PermanentLifePaymentInstruction) bool {
	subjects, ok := InspectContextualPermanentCollection(instruction.Subjects)
	if !ok || collectionReadsTargets(subjects.root) {
		return false
	}
	destructions, ok := InspectDirectDestruction(instruction.Body)
	if !ok || len(destructions) == 0 {
		return false
	}
	for _, destruction := range destructions {
		saved, ok := destruction.Subjects.(savedPermanentSet)
		if destruction.Subject != nil || !ok || saved.binding != instruction.Unpaid {
			return false
		}
	}
	return true
}
