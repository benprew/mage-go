package mage

// TurnAttackRuleInstruction retains a live collection rule until cleanup.
// Require saves Player at resolution. A restriction must not supply Player.
type TurnAttackRuleInstruction struct {
	Filter  PermanentFilter
	Player  PlayerRef
	Require bool
}

// Effect uses the shared combat rule and end-of-turn services.
// A missing required player or a player on a restriction returns nil.
func (instruction TurnAttackRuleInstruction) Effect() ResolvingEffect {
	if instruction.Require == nilDefinition(instruction.Player) {
		return nil
	}
	return &turnAttackRuleEffect{filter: instruction.Filter, player: instruction.Player, require: instruction.Require}
}

// InspectTurnAttackRuleInstruction reads one declaration without operand reads.
// Singular rules and other durations require separate contracts.
func InspectTurnAttackRuleInstruction(effect ResolvingEffect) (TurnAttackRuleInstruction, bool) {
	e, ok := effect.(*turnAttackRuleEffect)
	if !ok || e == nil {
		return TurnAttackRuleInstruction{}, false
	}
	instruction := TurnAttackRuleInstruction{Filter: e.filter, Player: e.player, Require: e.require}
	return instruction, instruction.Effect() != nil
}

func pureTurnAttackRule(instruction TurnAttackRuleInstruction) bool {
	if _, ok := inspectContextualPermanentCollectionFilter(instruction.Filter); !ok {
		return false
	}
	if !instruction.Require {
		return true
	}
	switch instruction.Player.(type) {
	case controllerRef, activePlayerRef:
		return true
	default:
		return false
	}
}
