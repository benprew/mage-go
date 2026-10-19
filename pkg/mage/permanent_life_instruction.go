package mage

// PermanentLifeInstruction retains life gain for one target's controller.
// Power or mana value is read when the life action runs, with exact visit LKI.
type PermanentLifeInstruction struct {
	Subject   *PermanentTargetBinding
	ManaValue bool
}

// Effect builds the shared information and life actions. Missing targets return nil.
func (instruction PermanentLifeInstruction) Effect() ResolvingEffect {
	if instruction.Subject == nil || instruction.Subject.targetDeclaration == nil {
		return nil
	}
	amount := PowerOfWithLKI(instruction.Subject)
	if instruction.ManaValue {
		amount = ManaValueOfWithLKI(instruction.Subject)
	}
	return (LifeInstruction{Subject: ControllerOfWithLKI(instruction.Subject), Amount: amount, Gain: true}).Effect()
}

// InspectPermanentLifeInstruction reads one exact native declaration.
// Both operands must use the same singular target. No information callbacks run.
func InspectPermanentLifeInstruction(effect ResolvingEffect) (PermanentLifeInstruction, bool) {
	instructions, ok := InspectLifeInstructions(effect)
	if !ok || len(instructions) != 1 {
		return PermanentLifeInstruction{}, false
	}
	life := instructions[0]
	controller, ok := life.Subject.(permanentInformationController)
	if !ok || !life.Gain {
		return PermanentLifeInstruction{}, false
	}
	target, ok := controller.subject.(*PermanentTargetBinding)
	if !ok || target == nil || target.targetDeclaration == nil {
		return PermanentLifeInstruction{}, false
	}
	amount, ok := life.Amount.(permanentInformationValue)
	if !ok || amount.subject != target || amount.toughness {
		return PermanentLifeInstruction{}, false
	}
	return PermanentLifeInstruction{Subject: target, ManaValue: amount.mana}, true
}
