package mage

import "github.com/benprew/mage-go/pkg/mage/core"

// TemporaryRampageInstruction retains separate rampage grants until end of turn.
type TemporaryRampageInstruction struct {
	Subject PermanentRef
	Amounts []int
}

type temporaryRampageEffect struct {
	*untilEffect
	instruction TemporaryRampageInstruction
}

// Effect uses the shared trigger grant and duration services.
// Missing subjects, empty lists and negative amounts return nil.
func (instruction TemporaryRampageInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Subject) || len(instruction.Amounts) == 0 {
		return nil
	}
	instruction.Amounts = append([]int(nil), instruction.Amounts...)
	modifications := make([]ContinuousModification, len(instruction.Amounts))
	for i, amount := range instruction.Amounts {
		if amount < 0 {
			return nil
		}
		trigger := WhenEvent(BecomesBlocked(This), RampageEffect(amount)).(*GenericTriggered)
		modifications[i] = TriggeredAbilityModification(instruction.Subject, trigger)
	}
	return &temporaryRampageEffect{untilEffect: &untilEffect{duration: core.EndOfTurn, modifications: modifications}, instruction: instruction}
}

// InspectTemporaryRampageInstruction copies the declaration without installing grants.
func InspectTemporaryRampageInstruction(effect ResolvingEffect) (TemporaryRampageInstruction, bool) {
	e, ok := effect.(*temporaryRampageEffect)
	if !ok || e == nil || e.untilEffect == nil {
		return TemporaryRampageInstruction{}, false
	}
	instruction := e.instruction
	instruction.Amounts = append([]int(nil), instruction.Amounts...)
	return instruction, true
}
