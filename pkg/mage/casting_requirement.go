package mage

// CastingRequirement checks a condition before casting choices and payment.
// It does not grant casting permission or add an instruction at resolution.
func CastingRequirement(condition StateCondition) *ActionDefinition {
	return NewAction(ActionSpell, WithCastingCondition(condition))
}
