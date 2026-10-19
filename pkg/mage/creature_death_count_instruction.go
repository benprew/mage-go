package mage

// CreatureDeathCountInstruction retains a creature-death count from one destruction.
// Result identifies the declaration. It does not contain a predicted count.
type CreatureDeathCountInstruction struct {
	Subjects         PermanentSet
	Result           *NumberBinding
	CannotRegenerate bool
}

// Effect uses shared simultaneous destruction and creature last known information.
// Missing subjects or an unnamed result return nil.
func (instruction CreatureDeathCountInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Subjects) || instruction.Result == nil || instruction.Result.name == "" {
		return nil
	}
	option := CreaturesDiedThisWayInto(instruction.Result)
	if instruction.CannotRegenerate {
		return DestroyEachWithoutRegeneration(instruction.Subjects, option)
	}
	return DestroyEach(instruction.Subjects, option)
}

// InspectCreatureDeathCountInstruction reads one producer without reading a count.
// Other result forms and duplicate options require separate contracts.
func InspectCreatureDeathCountInstruction(effect ResolvingEffect) (CreatureDeathCountInstruction, bool) {
	e, ok := effect.(*destroyEachEffect)
	if !ok || e == nil || e.invalidOption || e.optionCount != 1 || e.deathVisits != nil || e.graveyardMoves != nil {
		return CreatureDeathCountInstruction{}, false
	}
	instruction := CreatureDeathCountInstruction{Subjects: e.subjects, Result: e.creatureDeaths, CannotRegenerate: e.cannotRegenerate}
	return instruction, instruction.Effect() != nil
}

func inspectSavedNumberSum(value ResolutionValue) ([]*NumberBinding, bool) {
	if pureFixedOrXAmount(value) {
		return nil, true
	}
	if binding, ok := destructionResultRead(value); ok {
		return []*NumberBinding{binding}, true
	}
	sum, ok := value.(arithmeticValue)
	if !ok || sum.operation != valueAdd {
		return nil, false
	}
	left, ok := inspectSavedNumberSum(sum.left)
	if !ok {
		return nil, false
	}
	right, ok := inspectSavedNumberSum(sum.right)
	if !ok {
		return nil, false
	}
	return append(left, right...), true
}
