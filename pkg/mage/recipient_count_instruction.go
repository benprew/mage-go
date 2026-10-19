package mage

import "github.com/google/uuid"

// PlayerCollectionDamageInstruction retains a local player amount declaration.
// Execution reads all amounts before one simultaneous damage batch.
type PlayerCollectionDamageInstruction struct {
	Source   DamageSourceRef
	Subjects PlayerSet
	Member   *PlayerBinding
	Amount   ResolutionValue
}

// Effect builds the native action. Missing operands return nil.
func (instruction PlayerCollectionDamageInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Source) || nilDefinition(instruction.Subjects) || instruction.Member == nil || instruction.Member.name == "" || nilDefinition(instruction.Amount) {
		return nil
	}
	return DealDamageEachPlayer(instruction.Source, instruction.Subjects, instruction.Member, instruction.Amount)
}

// InspectPlayerCollectionDamageInstruction retains exact native operands.
// Other instructions need their own contracts. No operand callback runs.
func InspectPlayerCollectionDamageInstruction(effect ResolvingEffect) (PlayerCollectionDamageInstruction, bool) {
	native, ok := effect.(*playerDamageEachEffect)
	if !ok || native == nil {
		return PlayerCollectionDamageInstruction{}, false
	}
	instruction := PlayerCollectionDamageInstruction{native.source, native.subjects, native.member, native.amount}
	return instruction, instruction.Effect() != nil
}

// RecipientPermanentCountPredicate retains a count with one local player.
// The zero value is unknown. No game state is retained.
type RecipientPermanentCountPredicate struct {
	count  LivePermanentCountPredicate
	member *PlayerBinding
}

// InspectRecipientPermanentCount accepts native counts in a local player scope.
// Other saved players and unknown descendants reject without callbacks.
func InspectRecipientPermanentCount(value ResolutionValue, member *PlayerBinding) (RecipientPermanentCountPredicate, bool) {
	if member == nil || member.name == "" {
		return RecipientPermanentCountPredicate{}, false
	}
	count, ok := inspectLivePermanentCount(value, member)
	if !ok {
		return RecipientPermanentCountPredicate{}, false
	}
	return RecipientPermanentCountPredicate{count, member}, true
}

// Read counts current visits for one present recipient without changing frames.
// Layers must be current. Target bindings must be resolution-legal.
func (predicate RecipientPermanentCountPredicate) Read(ctx *EffectContext, recipient uuid.UUID) (int, bool) {
	if predicate.member == nil || ctx == nil || ctx.Game == nil || ctx.Game.GetPlayer(recipient) == nil {
		return 0, false
	}
	count := bindCountRecipient(predicate.count, recipient)
	return count.Read(ctx)
}

func bindCountRecipient(count LivePermanentCountPredicate, recipient uuid.UUID) LivePermanentCountPredicate {
	if count.inner != nil {
		inner := bindCountRecipient(*count.inner, recipient)
		count.inner = &inner
	}
	count.subjects.root = bindCollectionRecipient(count.subjects.root, recipient)
	return count
}

func bindCollectionRecipient(node *contextualCollectionNode, recipient uuid.UUID) *contextualCollectionNode {
	if node == nil {
		return nil
	}
	bound := *node
	if bound.player == collectionExplicitPlayer {
		bound.explicitPlayer = recipient
	}
	bound.hosts = bindCollectionRecipient(node.hosts, recipient)
	bound.children = make([]*contextualCollectionNode, len(node.children))
	for i, child := range node.children {
		bound.children[i] = bindCollectionRecipient(child, recipient)
	}
	return &bound
}
