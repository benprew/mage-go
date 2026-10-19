package mage

// NextEndStepInstruction retains a schedule for the next end step and its body.
// Execution uses the shared delayed frame and resolving controller.
type NextEndStepInstruction struct{ Body ResolvingEffect }

// Effect builds the native schedule. A missing body returns nil.
func (instruction NextEndStepInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Body) {
		return nil
	}
	return AtNext(EachEndStep, instruction.Body)
}

// InspectNextEndStepInstruction reads the body without creating a trigger.
// Consumers must validate every child before they use the declaration.
func InspectNextEndStepInstruction(effect ResolvingEffect) (NextEndStepInstruction, bool) {
	e, ok := effect.(*nextStepEffect)
	if !ok || e == nil || e.repeats || e.expiresAtCleanup || e.condition != nil || len(e.modes) != 0 || len(e.targets) != 0 || nilDefinition(e.body) {
		return NextEndStepInstruction{}, false
	}
	pattern, ok := e.pattern.(stepPattern)
	if !ok || pattern != EachEndStep {
		return NextEndStepInstruction{}, false
	}
	return NextEndStepInstruction{Body: e.body}, true
}

func inspectDelayedPermanentDestruction(effect ResolvingEffect) ([]spellPermanentInstruction, bool) {
	var children []ResolvingEffect
	switch e := effect.(type) {
	case *sequenceEffect:
		if e == nil {
			return nil, false
		}
		children = e.effects
	case *compositeEffect:
		if e == nil {
			return nil, false
		}
		children = e.effects
	default:
		if branch, ok := InspectPermanentBranchInstruction(effect); ok {
			target, ok := branch.Subject.(*PermanentTargetBinding)
			if !ok || target == nil {
				return nil, false
			}
			if _, ok := InspectPermanentTarget(target); !ok {
				return nil, false
			}
			if _, ok := inspectPermanentCollectionFilter(branch.Filter); !ok {
				return nil, false
			}
			children = []ResolvingEffect{branch.Matching}
			if branch.Nonmatching != nil {
				children = append(children, branch.Nonmatching)
			}
			out, ok := inspectDelayedPermanentDestruction(Sequence(children...))
			if !ok {
				return nil, false
			}
			return append([]spellPermanentInstruction{{deferred: true, subject: target}}, out...), true
		}
		instructions, ok := InspectDirectDestruction(effect)
		if !ok || len(instructions) == 0 {
			return nil, false
		}
		out := make([]spellPermanentInstruction, 0, len(instructions))
		for _, instruction := range instructions {
			target, ok := instruction.Subject.(*PermanentTargetBinding)
			if !ok || target == nil || instruction.Subjects != nil {
				return nil, false
			}
			if _, ok := InspectPermanentTarget(target); !ok {
				return nil, false
			}
			out = append(out, spellPermanentInstruction{deferred: true, subject: target})
		}
		return out, true
	}
	if len(children) == 0 {
		return nil, false
	}
	var out []spellPermanentInstruction
	for _, child := range children {
		nested, ok := inspectDelayedPermanentDestruction(child)
		if !ok {
			return nil, false
		}
		out = append(out, nested...)
	}
	return out, true
}
