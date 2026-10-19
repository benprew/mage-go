package mage

import "github.com/benprew/mage-go/pkg/mage/core"

// NextMainPhaseInstruction retains a one-shot schedule and its complete body.
// Execution uses the shared delayed frame and resolving controller.
type NextMainPhaseInstruction struct {
	Body ResolvingEffect
}

// Effect builds the native schedule. A missing body returns nil.
func (instruction NextMainPhaseInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Body) {
		return nil
	}
	return AtNext(YourMainPhase, instruction.Body)
}

// InspectNextMainPhaseInstruction retains the body without creating a trigger.
// Consumers must validate the complete body before they use its operands.
func InspectNextMainPhaseInstruction(effect ResolvingEffect) (NextMainPhaseInstruction, bool) {
	e, ok := effect.(*nextStepEffect)
	if !ok || e == nil || e.repeats || e.expiresAtCleanup || e.condition != nil || len(e.modes) != 0 || len(e.targets) != 0 || nilDefinition(e.body) {
		return NextMainPhaseInstruction{}, false
	}
	pattern, ok := e.pattern.(stepPattern)
	if !ok || pattern != YourMainPhase {
		return NextMainPhaseInstruction{}, false
	}
	return NextMainPhaseInstruction{Body: e.body}, true
}

func inspectNextMainMana(instruction NextMainPhaseInstruction) (spellPermanentInstruction, bool) {
	plan := spellPermanentInstruction{manaSubject: You}
	productions, ok := InspectManaProductionInstructions(instruction.Body)
	if !ok || len(productions) == 0 {
		return plan, false
	}
	for _, production := range productions {
		if _, ok := production.Subject.(controllerRef); !ok || production.Restriction != nil || production.Color == core.AnyColor {
			return plan, false
		}
		reads, ok := inspectDelayedSpellManaAmount(production.Amount)
		if !ok {
			return plan, false
		}
		plan.delayedManaReads = append(plan.delayedManaReads, reads...)
	}
	return plan, true
}

func inspectDelayedSpellManaAmount(amount ResolutionValue) ([]*SpellTargetBinding, bool) {
	if pureFixedOrXAmount(amount) {
		return nil, true
	}
	switch value := amount.(type) {
	case spellManaValue:
		target, ok := value.spell.(*SpellTargetBinding)
		if !ok || target == nil || !value.withLKI {
			return nil, false
		}
		if _, ok := InspectSpellTarget(target); !ok {
			return nil, false
		}
		return []*SpellTargetBinding{target}, true
	case arithmeticValue:
		if value.operation != valueAdd && value.operation != valueMultiply {
			return nil, false
		}
		left, leftOK := inspectDelayedSpellManaAmount(value.left)
		right, rightOK := inspectDelayedSpellManaAmount(value.right)
		if leftOK && rightOK {
			return append(left, right...), true
		}
	}
	return nil, false
}
