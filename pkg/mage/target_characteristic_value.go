package mage

import "github.com/google/uuid"

// TargetCharacteristicValuePredicate retains a native target power or mana value.
// It stores no game state. The zero value is unknown.
type TargetCharacteristicValuePredicate struct {
	value  permanentInformationValue
	target PermanentTargetPredicate
}

// InspectTargetCharacteristicValue accepts exact singular native target values.
// Unknown references and filters reject without callbacks or information reads.
func InspectTargetCharacteristicValue(value ResolutionValue) (TargetCharacteristicValuePredicate, bool) {
	information, ok := value.(permanentInformationValue)
	if !ok || information.toughness {
		return TargetCharacteristicValuePredicate{}, false
	}
	target, ok := information.subject.(*PermanentTargetBinding)
	if !ok {
		return TargetCharacteristicValuePredicate{}, false
	}
	predicate, ok := InspectPermanentTarget(target)
	if !ok {
		return TargetCharacteristicValuePredicate{}, false
	}
	return TargetCharacteristicValuePredicate{value: information, target: predicate}, true
}

func (predicate TargetCharacteristicValuePredicate) bindingsKnown(ctx *EffectContext) bool {
	if predicate.target.declaration == nil || ctx == nil || ctx.Game == nil || ctx.Game.GetPlayer(ctx.Controller) == nil {
		return false
	}
	values, ok := ctx.targetBindings[predicate.target.declaration]
	if !ok || len(values) > 1 {
		return false
	}
	return len(values) == 0 || values[0].id != uuid.Nil && values[0].incarnation != uuid.Nil
}

// Read uses current or last-known information for the exact bound visit.
// The frame must contain resolution-legal bindings. An empty binding supplies zero.
// Information does not grant permission to act on departed or phased objects.
// Layers must be current. No reference or filter callback runs.
func (predicate TargetCharacteristicValuePredicate) Read(ctx *EffectContext) (int, bool) {
	if !predicate.bindingsKnown(ctx) {
		return 0, false
	}
	if len(ctx.targetBindings[predicate.target.declaration]) == 0 {
		return 0, true
	}
	p, lki := readPermanentInformation(predicate.value.subject, ctx)
	if p == nil && lki == nil {
		return 0, false
	}
	return predicate.value.resolveValue(ctx), true
}
