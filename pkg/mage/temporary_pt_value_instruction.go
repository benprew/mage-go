package mage

import "github.com/benprew/mage-go/pkg/mage/core"

// PTValueAdjustment retains the values for one change in layer 7c.
type PTValueAdjustment struct {
	Power, Toughness ResolutionValue
}

// TemporaryPTValueInstruction captures all values before its layer applies.
type TemporaryPTValueInstruction struct {
	Subject     PermanentRef
	Adjustments []PTValueAdjustment
}

type temporaryPTValueEffect struct {
	*untilEffect
	instruction TemporaryPTValueInstruction
}

// Effect uses one shared layer installation until end of turn.
// Missing subjects, values or adjustments return nil.
func (instruction TemporaryPTValueInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Subject) || len(instruction.Adjustments) == 0 {
		return nil
	}
	instruction.Adjustments = append([]PTValueAdjustment(nil), instruction.Adjustments...)
	modifications := make([]ContinuousModification, len(instruction.Adjustments))
	for i, adjustment := range instruction.Adjustments {
		if nilDefinition(adjustment.Power) || nilDefinition(adjustment.Toughness) {
			return nil
		}
		modifications[i] = ModifyPT(instruction.Subject, adjustment.Power, adjustment.Toughness)
	}
	return &temporaryPTValueEffect{untilEffect: &untilEffect{duration: core.EndOfTurn, modifications: modifications}, instruction: instruction}
}

// InspectTemporaryPTValueInstruction copies operands without value reads.
// Consumers must validate the subject and each complete value tree.
func InspectTemporaryPTValueInstruction(effect ResolvingEffect) (TemporaryPTValueInstruction, bool) {
	e, ok := effect.(*temporaryPTValueEffect)
	if !ok || e == nil || e.untilEffect == nil {
		return TemporaryPTValueInstruction{}, false
	}
	instruction := e.instruction
	instruction.Adjustments = append([]PTValueAdjustment(nil), instruction.Adjustments...)
	return instruction, true
}

func inspectTemporaryPTValue(amount ResolutionValue) ([]*targetDeclaration, bool) {
	if pureFixedOrXAmount(amount) {
		return nil, true
	}
	var subject PermanentRef
	switch value := amount.(type) {
	case permanentInformationValue:
		subject = value.subject
	case permanentManaValue:
		subject = value.subject
	case arithmeticValue:
		if value.operation != valueSubtract && value.operation != valueMultiply {
			return nil, false
		}
		left, leftOK := inspectTemporaryPTValue(value.left)
		right, rightOK := inspectTemporaryPTValue(value.right)
		return append(left, right...), leftOK && rightOK
	default:
		return nil, false
	}
	target, ok := subject.(*PermanentTargetBinding)
	if !ok || target == nil {
		return nil, false
	}
	if _, ok := InspectPermanentTarget(target); !ok {
		return nil, false
	}
	return []*targetDeclaration{target.targetDeclaration}, true
}
