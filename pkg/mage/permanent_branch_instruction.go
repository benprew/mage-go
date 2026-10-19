package mage

// PermanentBranchInstruction retains one condition and its complete bodies.
// A nil Nonmatching body means that the condition has no alternative.
type PermanentBranchInstruction struct {
	Subject               PermanentRef
	Filter                PermanentFilter
	Matching, Nonmatching ResolvingEffect
	Negate                bool
}

// Effect uses the shared current and retained information service.
// Missing subjects or matching bodies return nil.
func (instruction PermanentBranchInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Subject) || nilDefinition(instruction.Matching) {
		return nil
	}
	if instruction.Nonmatching != nil && nilDefinition(instruction.Nonmatching) {
		return nil
	}
	return &permanentInformationBranch{subject: instruction.Subject, filter: instruction.Filter, body: instruction.Matching, otherwise: instruction.Nonmatching, hasElse: instruction.Nonmatching != nil, negate: instruction.Negate}
}

// InspectPermanentBranchInstruction reads the condition and both bodies without callbacks.
// Consumers must validate the complete filter, references and bodies before use.
func InspectPermanentBranchInstruction(effect ResolvingEffect) (PermanentBranchInstruction, bool) {
	e, ok := effect.(*permanentInformationBranch)
	if !ok || e == nil || e.controlHistory || nilDefinition(e.subject) || nilDefinition(e.body) || e.hasElse && nilDefinition(e.otherwise) {
		return PermanentBranchInstruction{}, false
	}
	instruction := PermanentBranchInstruction{Subject: e.subject, Filter: e.filter, Matching: e.body, Negate: e.negate}
	if e.hasElse {
		instruction.Nonmatching = e.otherwise
	}
	return instruction, true
}

func inspectTemporaryLayerBranch(instruction PermanentBranchInstruction) ([]spellPermanentInstruction, bool) {
	target, ok := instruction.Subject.(*PermanentTargetBinding)
	if !ok || target == nil {
		return nil, false
	}
	if _, ok := InspectPermanentTarget(target); !ok {
		return nil, false
	}
	if _, ok := inspectPermanentCollectionFilter(instruction.Filter); !ok {
		return nil, false
	}
	bodies := []ResolvingEffect{instruction.Matching}
	if instruction.Nonmatching != nil {
		bodies = append(bodies, instruction.Nonmatching)
	}
	children, ok := inspectSpellPermanentInstructionsUnchecked(bodies)
	if !ok || len(children) == 0 {
		return nil, false
	}
	for _, child := range children {
		if !child.temporaryLayer {
			return nil, false
		}
	}
	// Both paths can change characteristics before a later instruction.
	barrier := spellPermanentInstruction{temporaryLayer: true, subject: target, layerValueTargets: []*targetDeclaration{target.targetDeclaration}}
	return append([]spellPermanentInstruction{barrier}, children...), true
}
