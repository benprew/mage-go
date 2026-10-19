package mage

// TapOperationKind selects one native tap instruction.
type TapOperationKind uint8

const (
	TapOperation TapOperationKind = iota + 1
	UntapOperation
	MayTapOrUntapOperation
)

// TapInstruction retains one singular or simultaneous tap instruction.
// Exactly one subject field must be set. A choice requires a singular subject.
type TapInstruction struct {
	Subject   PermanentRef
	Subjects  PermanentSet
	Operation TapOperationKind
}

// Effect builds the native action. A malformed declaration returns nil.
func (instruction TapInstruction) Effect() ResolvingEffect {
	native := instruction.nativeEffect()
	if native == nil {
		return nil
	}
	return &tapInstructionEffect{native: native}
}

func (instruction TapInstruction) nativeEffect() ResolvingEffect {
	singular, plural := !nilDefinition(instruction.Subject), !nilDefinition(instruction.Subjects)
	if singular == plural {
		return nil
	}
	switch instruction.Operation {
	case TapOperation:
		if singular {
			return TapSubject(instruction.Subject)
		}
		return TapEach(instruction.Subjects)
	case UntapOperation:
		if singular {
			return UntapPermanent(instruction.Subject)
		}
		return UntapEach(instruction.Subjects)
	case MayTapOrUntapOperation:
		if singular {
			return MayTapOrUntap(instruction.Subject)
		}
	}
	return nil
}

// InspectTapInstructions reads complete native tap trees without callbacks.
// Result bindings and unknown children reject the whole tree.
func InspectTapInstructions(effect ResolvingEffect) ([]TapInstruction, bool) {
	switch effect := effect.(type) {
	case *tapInstructionEffect:
		if effect != nil {
			return InspectTapInstructions(effect.native)
		}
	case *sequenceEffect:
		if effect != nil {
			return inspectTapSequence(effect.effects)
		}
	case *compositeEffect:
		if effect != nil {
			return inspectTapSequence(effect.effects)
		}
	case *tapSubjectEffect:
		if effect != nil && effect.optionCount == 0 && !effect.invalidOption && effect.result == nil {
			return inspectTapLeaf(TapInstruction{Subject: effect.subject, Operation: TapOperation})
		}
	case *untapPermanentEffect:
		if effect != nil && !effect.requiresResult && effect.optionCount == 0 && !effect.invalidOption && effect.result == nil {
			return inspectTapLeaf(TapInstruction{Subject: effect.subject, Operation: UntapOperation})
		}
	case *tapEachEffect:
		if effect != nil {
			return inspectTapLeaf(TapInstruction{Subjects: effect.subjects, Operation: TapOperation})
		}
	case *untapEachEffect:
		if effect != nil {
			return inspectTapLeaf(TapInstruction{Subjects: effect.subjects, Operation: UntapOperation})
		}
	case *tapChoiceEffect:
		if effect != nil {
			return inspectTapLeaf(TapInstruction{Subject: effect.subject, Operation: MayTapOrUntapOperation})
		}
	}
	return nil, false
}

func inspectTapLeaf(instruction TapInstruction) ([]TapInstruction, bool) {
	if nilDefinition(instruction.Subject) && nilDefinition(instruction.Subjects) {
		return nil, false
	}
	return []TapInstruction{instruction}, true
}

func inspectTapSequence(effects []ResolvingEffect) ([]TapInstruction, bool) {
	var out []TapInstruction
	for _, effect := range effects {
		instructions, ok := InspectTapInstructions(effect)
		if !ok {
			return nil, false
		}
		out = append(out, instructions...)
	}
	return out, true
}

// The instruction boundary keeps legacy action and payment callers unchanged.
type tapInstructionEffect struct{ native ResolvingEffect }

func (effect *tapInstructionEffect) Text() string                 { return effect.native.Text() }
func (effect *tapInstructionEffect) Properties() EffectProperties { return effect.native.Properties() }
func (effect *tapInstructionEffect) Apply(ctx *EffectContext) error {
	if err := effect.native.Apply(ctx); err != nil {
		return err
	}
	ctx.Game.ApplyContinuousEffects()
	return nil
}
func (effect *tapInstructionEffect) inspectDefinition() definitionNode {
	return definitionNode{Role: "tap-instruction", children: []definitionEdge{{"native", effect.native}}}
}

func (effect *tapInstructionEffect) refersToSourcePermanent() bool {
	reference, ok := effect.native.(delayedSourcePermanentReference)
	return ok && reference.refersToSourcePermanent()
}

func (effect *tapInstructionEffect) stackPermanentBindings() []capturedPermanent {
	if reference, ok := effect.native.(stackPermanentBindingReferences); ok {
		return reference.stackPermanentBindings()
	}
	return nil
}
