package mage

import "github.com/benprew/mage-go/pkg/mage/core"

// TemporaryColorInstruction retains a collection color change until end of turn.
// An empty color list makes the captured members colorless.
type TemporaryColorInstruction struct {
	Subjects PermanentSet
	Colors   []core.Color
}

type temporaryColorEffect struct {
	*untilEffect
	instruction TemporaryColorInstruction
}

// Effect builds one layer 5 installation with the shared duration service.
// Missing subjects and invalid colors return nil.
func (instruction TemporaryColorInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Subjects) {
		return nil
	}
	for _, color := range instruction.Colors {
		if color < core.White || color > core.Green {
			return nil
		}
	}
	instruction.Colors = append([]core.Color(nil), instruction.Colors...)
	return &temporaryColorEffect{
		untilEffect: &untilEffect{duration: core.EndOfTurn, modifications: []ContinuousModification{SetEachColors(instruction.Subjects, instruction.Colors...)}},
		instruction: instruction,
	}
}

// InspectTemporaryColorInstruction copies a declaration without selecting members.
// Other layer forms and complete sequences need separate inspection contracts.
func InspectTemporaryColorInstruction(effect ResolvingEffect) (TemporaryColorInstruction, bool) {
	e, ok := effect.(*temporaryColorEffect)
	if !ok || e == nil || e.untilEffect == nil {
		return TemporaryColorInstruction{}, false
	}
	instruction := e.instruction
	instruction.Colors = append([]core.Color(nil), instruction.Colors...)
	return instruction, true
}
