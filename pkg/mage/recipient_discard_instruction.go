package mage

// DiscardUpToInstruction retains one optional discard batch and its result.
// The result identifies declarations; it does not retain game state.
type DiscardUpToInstruction struct {
	Subjects PlayerSet
	Maximum  ResolutionValue
	Result   *PlayerNumbersBinding
}

// Effect builds the shared discard action. Missing declarations return nil.
func (instruction DiscardUpToInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Subjects) || nilDefinition(instruction.Maximum) || !validPlayerNumbers(instruction.Result) {
		return nil
	}
	return DiscardUpToEach(instruction.Subjects, instruction.Maximum, instruction.Result)
}

func validPlayerNumbers(binding *PlayerNumbersBinding) bool {
	return binding != nil && binding.members[0] != nil && binding.members[1] != nil && binding.members[0] != binding.members[1] && binding.members[0].name != "" && binding.members[1].name != ""
}

// InspectDiscardUpToInstruction retains one complete native discard producer.
// Inspection does not run choices, replacements or operand callbacks.
func InspectDiscardUpToInstruction(effect ResolvingEffect) (DiscardUpToInstruction, bool) {
	native, ok := effect.(*discardUpToEachEffect)
	if !ok || native == nil {
		return DiscardUpToInstruction{}, false
	}
	instruction := DiscardUpToInstruction{native.subjects, native.maximum, native.result}
	return instruction, instruction.Effect() != nil
}

// RecipientDiscardAmount retains fixed subtraction from one player's result.
// It does not read or predict the saved count.
type RecipientDiscardAmount struct {
	Base   int
	Result *PlayerNumbersBinding
}

// InspectRecipientDiscardAmount accepts fixed minus a native local player count.
// The saved player must use the exact supplied member declaration.
func InspectRecipientDiscardAmount(value ResolutionValue, member *PlayerBinding) (RecipientDiscardAmount, bool) {
	if member == nil || member.name == "" {
		return RecipientDiscardAmount{}, false
	}
	arithmetic, ok := value.(arithmeticValue)
	if !ok || arithmetic.operation != valueSubtract {
		return RecipientDiscardAmount{}, false
	}
	left := arithmetic.left
	if legacy, ok := left.(legacyResolutionValue); ok {
		fixed, ok := legacy.value.(fixedValue)
		if !ok {
			return RecipientDiscardAmount{}, false
		}
		left = fixed
	}
	base, ok := left.(fixedValue)
	if !ok {
		return RecipientDiscardAmount{}, false
	}
	count, ok := arithmetic.right.(playerNumberValue)
	if !ok || !validPlayerNumbers(count.binding) {
		return RecipientDiscardAmount{}, false
	}
	player, ok := count.player.(savedPlayerRef)
	if !ok || player.binding != member {
		return RecipientDiscardAmount{}, false
	}
	return RecipientDiscardAmount{base.n, count.binding}, true
}
