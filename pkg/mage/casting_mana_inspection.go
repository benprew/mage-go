package mage

// castingManaReads accepts only native scalar trees. It never reads game state.
func castingManaReads(amount ResolutionValue) ([]*NumberBinding, bool) {
	if pureFixedOrXAmount(amount) {
		return nil, true
	}
	switch amount := amount.(type) {
	case savedNumberValue:
		if amount.binding == nil || amount.binding.name == "" {
			return nil, false
		}
		return []*NumberBinding{amount.binding}, true
	case arithmeticValue:
		if amount.operation != valueAdd && amount.operation != valueMultiply {
			return nil, false
		}
		left, ok := castingManaReads(amount.left)
		if !ok {
			return nil, false
		}
		right, ok := castingManaReads(amount.right)
		if !ok {
			return nil, false
		}
		return append(left, right...), true
	default:
		return nil, false
	}
}

// validCastingManaReads requires the exact paid producer before resolution.
// A result map alone cannot establish which operation supplied a number.
func validCastingManaReads(obj *StackObject, instructions []spellPermanentInstruction) bool {
	reads := make(map[*NumberBinding]bool)
	for _, instruction := range instructions {
		for _, binding := range instruction.castingManaReads {
			reads[binding] = true
		}
	}
	if len(reads) == 0 {
		return true
	}
	definition := obj.spellTextDefinition
	if nilDefinition(definition) {
		definition = obj.Card
	}
	base, ok := definition.(*BaseCard)
	if !ok || base == nil || len(base.additionalCosts) != 1 {
		return false
	}
	cost, ok := base.additionalCosts[0].(*sacrificeMatchingCost)
	if !ok || cost == nil || cost.count != 1 || !cost.captureManaValue || cost.paidManaValue == nil || cost.text == "" ||
		cost.captureStats || cost.captureSubtype || cost.captureCard || cost.capturePermanent || cost.captureTotalPower {
		return false
	}
	for binding := range reads {
		value, present := obj.capturedNumbers[binding]
		if binding != cost.paidManaValue || !present || value < 0 {
			return false
		}
	}
	for _, instruction := range instructions {
		if reads[instruction.resultWrite] {
			return false
		}
		if instruction.discardWrite != nil {
			for _, member := range instruction.discardWrite.members {
				if reads[member] {
					return false
				}
			}
		}
	}
	return true
}
