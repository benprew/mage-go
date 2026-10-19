package mage

// ForcedManaLossInstruction retains forced activations and the following loss.
// Result is optional. A saved player supplies both operations.
type ForcedManaLossInstruction struct {
	Subject PlayerRef
	Filter  PermanentFilter
	Result  *LostManaBinding
}

// Effect builds the shared activation and mana loss sequence.
// Missing operands or an unnamed result return nil.
func (instruction ForcedManaLossInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Subject) || instruction.Filter.IsZero() || instruction.Result != nil && instruction.Result.name == "" {
		return nil
	}
	saved := BindPlayer("forced mana player")
	ref := FromPlayer(saved)
	loss := LoseUnspentManaFor(ref)
	if instruction.Result != nil {
		loss = LoseUnspentManaInto(ref, instruction.Result)
	}
	body := &sequenceEffect{effects: []ResolvingEffect{SnapshotPlayer(instruction.Subject, saved), ActivateManaAbilityOfEach(ref, instruction.Filter), loss}}
	return &forcedManaLossEffect{instruction: instruction, body: body}
}

type forcedManaLossEffect struct {
	instruction ForcedManaLossInstruction
	body        *sequenceEffect
}

func (e *forcedManaLossEffect) Text() string                   { return e.body.Text() }
func (e *forcedManaLossEffect) Properties() EffectProperties   { return e.body.Properties() }
func (e *forcedManaLossEffect) Apply(ctx *EffectContext) error { return e.body.Apply(ctx) }
func (e *forcedManaLossEffect) inspectDefinition() definitionNode {
	n := definitionNode{Role: "forced-mana-loss", Text: e.Text(), children: []definitionEdge{{"subject", e.instruction.Subject}}}
	if e.instruction.Result != nil {
		n.bindingWrite = e.instruction.Result
		n.children = append(n.children, definitionEdge{"result", e.instruction.Result})
	}
	return n
}
func (e *forcedManaLossEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.instruction.Subject)
}

// InspectForcedManaLossInstruction retains native operands without activations.
// Consumers must validate the player and filter before use.
func InspectForcedManaLossInstruction(effect ResolvingEffect) (ForcedManaLossInstruction, bool) {
	e, ok := effect.(*forcedManaLossEffect)
	if !ok || e == nil {
		return ForcedManaLossInstruction{}, false
	}
	return e.instruction, e.instruction.Effect() != nil
}

// LostManaInstruction retains a native saved loss or transfer.
// Restore transfers the saved units with all restrictions and source metadata.
type LostManaInstruction struct {
	Subject PlayerRef
	Result  *LostManaBinding
	Restore bool
}

// Effect builds shared mana removal or transfer. Missing operands return nil.
func (instruction LostManaInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Subject) || instruction.Result == nil || instruction.Result.name == "" {
		return nil
	}
	if instruction.Restore {
		return AddLostManaFor(instruction.Subject, instruction.Result)
	}
	return LoseUnspentManaInto(instruction.Subject, instruction.Result)
}

// InspectLostManaInstruction retains one native loss or transfer declaration.
// Inspection does not read saved mana or change any mana pool.
func InspectLostManaInstruction(effect ResolvingEffect) (LostManaInstruction, bool) {
	e, ok := effect.(*lostManaEffect)
	if !ok || e == nil {
		return LostManaInstruction{}, false
	}
	instruction := LostManaInstruction{Subject: e.player, Result: e.result, Restore: e.restore}
	return instruction, instruction.Effect() != nil
}
