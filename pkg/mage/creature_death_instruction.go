package mage

import "github.com/benprew/mage-go/pkg/mage/core"

// CreatureDeathInstruction retains exact creature visits from one destruction.
type CreatureDeathInstruction struct {
	Subjects         PermanentSet
	Result           *PermanentSetBinding
	CannotRegenerate bool
}

// Effect builds the shared destruction action. Missing operands return nil.
func (instruction CreatureDeathInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Subjects) || instruction.Result == nil || instruction.Result.name == "" {
		return nil
	}
	if instruction.CannotRegenerate {
		return DestroyEachWithoutRegeneration(instruction.Subjects, CreatureDeathsInto(instruction.Result))
	}
	return DestroyEach(instruction.Subjects, CreatureDeathsInto(instruction.Result))
}

// InspectCreatureDeathInstruction reads one exact-visit producer without selection.
func InspectCreatureDeathInstruction(effect ResolvingEffect) (CreatureDeathInstruction, bool) {
	e, ok := effect.(*destroyEachEffect)
	if !ok || e == nil || e.invalidOption || e.optionCount != 1 || e.creatureDeaths != nil || e.graveyardMoves != nil {
		return CreatureDeathInstruction{}, false
	}
	instruction := CreatureDeathInstruction{e.subjects, e.deathVisits, e.cannotRegenerate}
	if instruction.Effect() == nil {
		return CreatureDeathInstruction{}, false
	}
	return instruction, true
}

// CreatureDeathReturnInstruction retains one owner-controlled creature-card return
// for each saved death. Consumers must validate the producer and player bindings.
type CreatureDeathReturnInstruction struct {
	Result    *PermanentSetBinding
	Member    *PermanentBinding
	Graveyard PlayerRef
}

// Effect builds the shared choice and simultaneous entry operation.
func (instruction CreatureDeathReturnInstruction) Effect() ResolvingEffect {
	if instruction.Result == nil || instruction.Result.name == "" || instruction.Member == nil || instruction.Member.name == "" || nilDefinition(instruction.Graveyard) {
		return nil
	}
	return ReturnChosenGraveyardCardsForEachPermanent(instruction.Result, instruction.Member, instruction.Graveyard, HasCardTypeCardFilter(core.TypeCreature))
}

// InspectCreatureDeathReturnInstruction reads the declaration without choices.
// Opaque filters and other card types require separate inspection contracts.
func InspectCreatureDeathReturnInstruction(effect ResolvingEffect) (CreatureDeathReturnInstruction, bool) {
	e, ok := effect.(*deathResultReturn)
	if !ok || e == nil || e.filter.IsZero() {
		return CreatureDeathReturnInstruction{}, false
	}
	filter := e.filter.characteristics
	if filter == nil || filter.operation != cardCharacteristicType || filter.kind != core.TypeCreature || len(filter.children) != 0 {
		return CreatureDeathReturnInstruction{}, false
	}
	instruction := CreatureDeathReturnInstruction{e.subjects, e.member, e.graveyard}
	if instruction.Effect() == nil {
		return CreatureDeathReturnInstruction{}, false
	}
	return instruction, true
}

func inspectCreatureDeathReturnPlayer(instruction CreatureDeathReturnInstruction) ([]*targetDeclaration, bool) {
	switch player := instruction.Graveyard.(type) {
	case controllerRef:
		return nil, true
	case lastBlockedController:
		attacker, ok := player.attacker.(savedPermanentRef)
		if !ok || attacker.binding != instruction.Member {
			return nil, false
		}
		blocker, ok := player.blocker.(*PermanentTargetBinding)
		if !ok || blocker == nil {
			return nil, false
		}
		if _, ok := InspectPermanentTarget(blocker); !ok {
			return nil, false
		}
		return []*targetDeclaration{blocker.targetDeclaration}, true
	}
	return nil, false
}
