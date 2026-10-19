package mage

// OrderedDestructionPredicate retains complete simultaneous collection actions.
// It has no selected objects. The zero value cannot establish certainty.
type OrderedDestructionPredicate struct {
	instructions []orderedDestructionInstruction
	valid        bool
}

type orderedDestructionInstruction struct {
	subjects         ContextualPermanentCollectionPredicate
	cannotRegenerate bool
}

// InspectOrderedDestruction retains a complete direct collection destruction tree.
// Singular subjects, result writes and unknown descendants reject the whole tree.
// Inspection does not execute effects or call operand callbacks.
func InspectOrderedDestruction(effect ResolvingEffect) (OrderedDestructionPredicate, bool) {
	instructions, ok := InspectDirectDestruction(effect)
	if !ok {
		return OrderedDestructionPredicate{}, false
	}
	plan := OrderedDestructionPredicate{valid: true}
	for _, instruction := range instructions {
		if instruction.Subject != nil {
			return OrderedDestructionPredicate{}, false
		}
		subjects, ok := InspectContextualPermanentCollection(instruction.Subjects)
		if !ok {
			return OrderedDestructionPredicate{}, false
		}
		plan.instructions = append(plan.instructions, orderedDestructionInstruction{
			subjects: subjects, cannotRegenerate: instruction.CannotRegenerate,
		})
	}
	return plan, true
}

// Certain checks exact watched visits without executing the retained actions.
// Supply current layers and the action's resolving controller. Target legality
// and whether the source resolves are separate caller contracts.
// A true first result proves a watched visit must move to the graveyard in a
// direct destruction group. False, true means the complete query found no proof.
// False, false means the outcome is unknown. A group that can propose destruction
// stops later analysis because replacements can change state without a move.
// The query does not consume shields, choose replacements or change game state.
func (p OrderedDestructionPredicate) Certain(ctx *EffectContext, watched []PermanentVisit) (certain, known bool) {
	if !p.valid || ctx == nil || ctx.Game == nil || ctx.Game.GetPlayer(ctx.Controller) == nil {
		return false, false
	}
	for _, instruction := range p.instructions {
		if !instruction.subjects.root.bindingsKnown(ctx) {
			return false, false
		}
	}
	visits := make(map[PermanentVisit]bool, len(watched))
	for _, visit := range watched {
		visits[visit] = true
	}
	for i, instruction := range p.instructions {
		selected := instruction.subjects.Select(ctx)
		proof := false
		for _, visit := range selected {
			destinations, known := ctx.Game.InspectDestructionDestinations(visit, instruction.cannotRegenerate)
			if !known {
				return false, false
			}
			if visits[visit] && destinations == DestructionToGraveyard {
				proof = true
			}
		}
		if proof {
			return true, true
		}
		if i+1 < len(p.instructions) {
			mayChange, known := ctx.Game.InspectDestructionStateChange(selected)
			if !known || mayChange {
				return false, false
			}
		}
	}
	return false, true
}
