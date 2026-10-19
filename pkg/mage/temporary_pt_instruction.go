package mage

import "github.com/benprew/mage-go/pkg/mage/core"

// PTAdjustment retains one fixed change in layer 7c.
type PTAdjustment struct {
	Power, Toughness int
}

// TemporaryPTInstruction retains fixed adjustments until end of turn.
// Exactly one subject form is required. All adjustments share one installation.
type TemporaryPTInstruction struct {
	Subject     PermanentRef
	Subjects    PermanentSet
	Adjustments []PTAdjustment
}

type temporaryPTEffect struct {
	*untilEffect
	instruction TemporaryPTInstruction
}

// Effect builds the instruction with the shared layer and duration services.
// Missing subjects, conflicting subjects and empty adjustments return nil.
func (instruction TemporaryPTInstruction) Effect() ResolvingEffect {
	singular, plural := !nilDefinition(instruction.Subject), !nilDefinition(instruction.Subjects)
	if singular == plural || len(instruction.Adjustments) == 0 {
		return nil
	}
	instruction.Adjustments = append([]PTAdjustment(nil), instruction.Adjustments...)
	modifications := make([]ContinuousModification, len(instruction.Adjustments))
	for i, adjustment := range instruction.Adjustments {
		if singular {
			modifications[i] = ModifyPT(instruction.Subject, Fixed(adjustment.Power), Fixed(adjustment.Toughness))
		} else {
			modifications[i] = ModifyEachPT(instruction.Subjects, Fixed(adjustment.Power), Fixed(adjustment.Toughness))
		}
	}
	return &temporaryPTEffect{untilEffect: &untilEffect{duration: core.EndOfTurn, modifications: modifications}, instruction: instruction}
}

// InspectTemporaryPTInstruction reads one native declaration without callbacks.
// Other layer forms and complete sequences require separate inspection contracts.
func InspectTemporaryPTInstruction(effect ResolvingEffect) (TemporaryPTInstruction, bool) {
	e, ok := effect.(*temporaryPTEffect)
	if !ok || e == nil || e.untilEffect == nil {
		return TemporaryPTInstruction{}, false
	}
	instruction := e.instruction
	instruction.Adjustments = append([]PTAdjustment(nil), instruction.Adjustments...)
	return instruction, true
}
