package mage

// ManaLossInstruction retains one native unspent mana removal action.
// Saved mana results use separate declarations.
type ManaLossInstruction struct {
	Subject PlayerRef
}

// Effect builds the native action. A missing subject returns nil.
func (instruction ManaLossInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Subject) {
		return nil
	}
	return LoseUnspentManaFor(instruction.Subject)
}

// InspectManaLossInstructions reads complete native mana removal trees.
// It does not resolve players or clear mana. Unknown children reject the tree.
func InspectManaLossInstructions(effect ResolvingEffect) ([]ManaLossInstruction, bool) {
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
	case *loseUnspentManaEffect:
		if effect != nil {
			instruction := ManaLossInstruction{Subject: effect.player}
			if instruction.Effect() != nil {
				return []ManaLossInstruction{instruction}, true
			}
		}
		return nil, false
	default:
		return nil, false
	}
	var out []ManaLossInstruction
	for _, child := range children {
		instructions, ok := InspectManaLossInstructions(child)
		if !ok {
			return nil, false
		}
		out = append(out, instructions...)
	}
	return out, true
}
