package mage

import "github.com/benprew/mage-go/pkg/mage/core"

// CombatSourcePreventionInstruction retains combat prevention from one source.
// The shared replacement service captures its visit until end of turn.
type CombatSourcePreventionInstruction struct{ Subject PermanentRef }

// Effect builds the installation. A missing source returns nil.
func (instruction CombatSourcePreventionInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Subject) {
		return nil
	}
	return PreventCombatDamageByThisTurn(instruction.Subject)
}

// InspectCombatSourcePreventionInstruction reads the complete native installation.
// Other durations, source sets and recipient prevention need separate contracts.
func InspectCombatSourcePreventionInstruction(effect ResolvingEffect) (CombatSourcePreventionInstruction, bool) {
	e, ok := effect.(*combatSourcePreventionInstallation)
	if !ok || e == nil || e.duration != core.EndOfTurn || nilDefinition(e.subject) || e.subjects != nil {
		return CombatSourcePreventionInstruction{}, false
	}
	return CombatSourcePreventionInstruction{Subject: e.subject}, true
}
