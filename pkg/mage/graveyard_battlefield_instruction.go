package mage

// GraveyardBattlefieldInstruction retains a mandatory, normal graveyard entry.
// Execution uses the shared graveyard movement and entry replacement services.
type GraveyardBattlefieldInstruction struct {
	Subject    CardRef
	Controller PlayerRef
}

// Effect builds the native return. Missing operands return nil.
func (instruction GraveyardBattlefieldInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Subject) || nilDefinition(instruction.Controller) {
		return nil
	}
	return ReturnCardToBattlefield(instruction.Subject, instruction.Controller)
}

// InspectGraveyardBattlefieldInstruction reads a normal entry without moving cards.
// Choices, saved results and entry changes require separate inspection contracts.
func InspectGraveyardBattlefieldInstruction(effect ResolvingEffect) (GraveyardBattlefieldInstruction, bool) {
	e, ok := effect.(*returnCardToBattlefieldEffect)
	if !ok || e == nil || nilDefinition(e.card) || nilDefinition(e.controller) || e.optional || e.saveResult || e.entry != (battlefieldEntryOptions{}) {
		return GraveyardBattlefieldInstruction{}, false
	}
	return GraveyardBattlefieldInstruction{Subject: e.card, Controller: e.controller}, true
}
