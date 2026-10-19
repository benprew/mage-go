package mage

import "github.com/benprew/mage-go/pkg/mage/core"

// ObjectColorInstruction retains one fixed color for a spell or permanent target.
type ObjectColorInstruction struct {
	Subject *SpellOrPermanentTargetBinding
	Color   core.Color
}

// Effect uses the shared color and target services for the selected object.
// Invalid declarations return nil.
func (instruction ObjectColorInstruction) Effect() ResolvingEffect {
	if _, ok := InspectObjectTarget(instruction.Subject); !ok || instruction.Color < core.White || instruction.Color > core.Green {
		return nil
	}
	return SetObjectColor(instruction.Subject, instruction.Color)
}

// InspectObjectColorInstruction reads a complete declaration without changing colors.
func InspectObjectColorInstruction(effect ResolvingEffect) (ObjectColorInstruction, bool) {
	e, ok := effect.(*objectTargetColorEffect)
	if !ok || e == nil {
		return ObjectColorInstruction{}, false
	}
	instruction := ObjectColorInstruction{Subject: e.object, Color: e.color}
	if _, ok := InspectObjectTarget(instruction.Subject); !ok || instruction.Color < core.White || instruction.Color > core.Green {
		return ObjectColorInstruction{}, false
	}
	return instruction, true
}
