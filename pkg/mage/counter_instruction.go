package mage

// CounterInstruction retains one native spell counter action.
// Ability counters and saved results have separate contracts.
type CounterInstruction struct {
	Subject StackObjectRef
}

// Effect builds the native action. A missing subject returns nil.
func (instruction CounterInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Subject) {
		return nil
	}
	return Counter(instruction.Subject)
}

// InspectCounterInstructions reads complete native spell counter trees.
// It does not resolve references, remove spells or execute callbacks.
func InspectCounterInstructions(effect ResolvingEffect) ([]CounterInstruction, bool) {
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
	case *counterReferencedEffect:
		if effect != nil && !effect.ability && !nilDefinition(effect.spell) {
			return []CounterInstruction{{Subject: effect.spell}}, true
		}
		return nil, false
	default:
		return nil, false
	}
	var out []CounterInstruction
	for _, child := range children {
		instructions, ok := InspectCounterInstructions(child)
		if !ok {
			return nil, false
		}
		out = append(out, instructions...)
	}
	return out, true
}
