package mage

import "github.com/benprew/mage-go/pkg/mage/core"

// TemporaryKeywordInstruction retains keyword grants until end of turn.
// Exactly one subject form is required. All keyword grants share one installation.
type TemporaryKeywordInstruction struct {
	Subject  PermanentRef
	Subjects PermanentSet
	Keywords []core.Attr
}

type temporaryKeywordEffect struct {
	*untilEffect
	instruction TemporaryKeywordInstruction
}

// Effect builds the instruction with the shared layer and duration services.
// Invalid subjects, empty lists and attributes outside the keyword range return nil.
func (instruction TemporaryKeywordInstruction) Effect() ResolvingEffect {
	singular, plural := !nilDefinition(instruction.Subject), !nilDefinition(instruction.Subjects)
	if singular == plural || len(instruction.Keywords) == 0 {
		return nil
	}
	for _, keyword := range instruction.Keywords {
		if keyword < core.Flying || keyword > core.Ward || !core.IsAbilityAttr(keyword) {
			return nil
		}
	}
	instruction.Keywords = append([]core.Attr(nil), instruction.Keywords...)
	modifications := make([]ContinuousModification, len(instruction.Keywords))
	for i, keyword := range instruction.Keywords {
		if singular {
			modifications[i] = GrantKeywordModification(instruction.Subject, keyword)
		} else {
			modifications[i] = GrantEachKeyword(instruction.Subjects, keyword)
		}
	}
	return &temporaryKeywordEffect{untilEffect: &untilEffect{duration: core.EndOfTurn, modifications: modifications}, instruction: instruction}
}

// InspectTemporaryKeywordInstruction reads one native declaration without callbacks.
// Other layer forms and complete sequences require separate inspection contracts.
func InspectTemporaryKeywordInstruction(effect ResolvingEffect) (TemporaryKeywordInstruction, bool) {
	e, ok := effect.(*temporaryKeywordEffect)
	if !ok || e == nil || e.untilEffect == nil {
		return TemporaryKeywordInstruction{}, false
	}
	instruction := e.instruction
	instruction.Keywords = append([]core.Attr(nil), instruction.Keywords...)
	return instruction, true
}
