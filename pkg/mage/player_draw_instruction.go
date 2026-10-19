package mage

// PlayerDrawInstruction retains a shared amount for a player collection.
// Execution reads the amount once, then draws in active player order.
type PlayerDrawInstruction struct {
	Subjects PlayerSet
	Amount   ResolutionValue
}

// Effect builds the shared draw action. Missing operands return nil.
func (instruction PlayerDrawInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Subjects) || nilDefinition(instruction.Amount) {
		return nil
	}
	return DrawEach(instruction.Subjects, instruction.Amount)
}

// InspectPlayerDrawInstruction reads one native declaration without callbacks.
// Singular draws and result bindings use separate declarations.
func InspectPlayerDrawInstruction(effect ResolvingEffect) (PlayerDrawInstruction, bool) {
	native, ok := effect.(*drawEachEffect)
	if !ok || native == nil {
		return PlayerDrawInstruction{}, false
	}
	instruction := PlayerDrawInstruction{Subjects: native.subjects, Amount: native.amount}
	return instruction, instruction.Effect() != nil
}
