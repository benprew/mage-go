package mage

// GlobalCombatPreventionInstruction retains all combat prevention until cleanup.
// It has no source, recipient, amount or choice operands.
type GlobalCombatPreventionInstruction struct{}

// Effect uses the shared combat damage prevention service.
func (GlobalCombatPreventionInstruction) Effect() ResolvingEffect {
	return PreventAllCombatDamage()
}

// InspectGlobalCombatPreventionInstruction reads one complete native declaration.
// It does not install prevention or inspect different prevention forms.
func InspectGlobalCombatPreventionInstruction(effect ResolvingEffect) (GlobalCombatPreventionInstruction, bool) {
	e, ok := effect.(*preventAllCombatDamageEffect)
	return GlobalCombatPreventionInstruction{}, ok && e != nil
}
