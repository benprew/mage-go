package mage

import "github.com/benprew/mage-go/pkg/mage/core"

// TemporaryMixedLayerInstruction retains keyword and P/T changes for one subject.
// All values are saved before the shared layer installation applies.
type TemporaryMixedLayerInstruction struct {
	Subject     PermanentRef
	Keywords    []core.Attr
	Adjustments []PTValueAdjustment
}

type temporaryMixedLayerEffect struct {
	*untilEffect
	instruction TemporaryMixedLayerInstruction
}

// Effect builds one installation with layer 6 and 7c parts until end of turn.
// Missing operands, empty lists and invalid keywords return nil.
func (instruction TemporaryMixedLayerInstruction) Effect() ResolvingEffect {
	keywords := (TemporaryKeywordInstruction{Subject: instruction.Subject, Keywords: instruction.Keywords}).Effect()
	values := (TemporaryPTValueInstruction{Subject: instruction.Subject, Adjustments: instruction.Adjustments}).Effect()
	if keywords == nil || values == nil {
		return nil
	}
	keywordEffect := keywords.(*temporaryKeywordEffect)
	valueEffect := values.(*temporaryPTValueEffect)
	instruction.Keywords = keywordEffect.instruction.Keywords
	instruction.Adjustments = valueEffect.instruction.Adjustments
	modifications := append([]ContinuousModification(nil), keywordEffect.modifications...)
	modifications = append(modifications, valueEffect.modifications...)
	return &temporaryMixedLayerEffect{untilEffect: &untilEffect{duration: core.EndOfTurn, modifications: modifications}, instruction: instruction}
}

// InspectTemporaryMixedLayerInstruction copies operands without value reads.
// Consumers must validate the subject and each complete value tree.
func InspectTemporaryMixedLayerInstruction(effect ResolvingEffect) (TemporaryMixedLayerInstruction, bool) {
	e, ok := effect.(*temporaryMixedLayerEffect)
	if !ok || e == nil || e.untilEffect == nil {
		return TemporaryMixedLayerInstruction{}, false
	}
	instruction := e.instruction
	instruction.Keywords = append([]core.Attr(nil), instruction.Keywords...)
	instruction.Adjustments = append([]PTValueAdjustment(nil), instruction.Adjustments...)
	return instruction, true
}
