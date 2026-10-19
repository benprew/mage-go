package mage

import "github.com/google/uuid"

// CollectionDamageInstruction retains one simultaneous native damage action.
// Division and result bindings require separate declarations.
type CollectionDamageInstruction struct {
	Source     DamageSourceRef
	Recipients DamageRecipientSet
	Amount     ResolutionValue
}

// Effect builds the native action. A missing operand returns nil.
func (instruction CollectionDamageInstruction) Effect() ResolvingEffect {
	if nilDefinition(instruction.Source) || nilDefinition(instruction.Recipients) || nilDefinition(instruction.Amount) {
		return nil
	}
	return DealDamageEach(instruction.Source, instruction.Recipients, instruction.Amount)
}

// InspectCollectionDamageInstructions reads complete native trees without callbacks.
// Division, result bindings and unknown children reject the whole tree.
func InspectCollectionDamageInstructions(effect ResolvingEffect) ([]CollectionDamageInstruction, bool) {
	switch effect := effect.(type) {
	case *sequenceEffect:
		if effect != nil {
			return inspectCollectionDamageSequence(effect.effects)
		}
	case *compositeEffect:
		if effect != nil {
			return inspectCollectionDamageSequence(effect.effects)
		}
	case *damageEachEffect:
		if effect != nil && !effect.evenDivision && !effect.invalidOption && effect.optionCount == 0 && effect.resultsCount == 0 && effect.total == nil && effect.results == nil {
			instruction := CollectionDamageInstruction{Source: effect.source, Recipients: effect.recipients, Amount: effect.amount}
			if instruction.Effect() != nil {
				return []CollectionDamageInstruction{instruction}, true
			}
		}
	}
	return nil, false
}

func inspectCollectionDamageSequence(effects []ResolvingEffect) ([]CollectionDamageInstruction, bool) {
	var out []CollectionDamageInstruction
	for _, effect := range effects {
		instructions, ok := InspectCollectionDamageInstructions(effect)
		if !ok {
			return nil, false
		}
		out = append(out, instructions...)
	}
	return out, true
}

// DamageRecipientCollectionPredicate retains a complete native recipient collection.
// It has no chosen members. The zero value cannot select recipients.
type DamageRecipientCollectionPredicate struct {
	permanents ContextualPermanentCollectionPredicate
	players    PlayerCollectionPredicate
	targets    *targetDeclaration
}

// InspectDamageRecipientCollection reads complete native player and permanent sets.
// Player sets use the shared native player collection contract. Permanent sets
// use the shared contextual collection contract. Native damage target groups
// require a legal target frame. Unknown children reject the query.
func InspectDamageRecipientCollection(recipients DamageRecipientSet) (DamageRecipientCollectionPredicate, bool) {
	var players PlayerSet
	var permanents ContextualPermanentCollectionPredicate
	switch set := recipients.(type) {
	case *DamageTargetsBinding:
		predicate, ok := InspectDamageTargets(set)
		if !ok {
			return DamageRecipientCollectionPredicate{}, false
		}
		return DamageRecipientCollectionPredicate{targets: predicate.declaration}, true
	case playerDamageRecipients:
		players = set.players
	case mixedDamageRecipients:
		var ok bool
		permanents, ok = InspectContextualPermanentCollection(set.permanents)
		if !ok {
			return DamageRecipientCollectionPredicate{}, false
		}
		players = set.players
	default:
		return DamageRecipientCollectionPredicate{}, false
	}
	predicate, ok := InspectPlayerCollection(players)
	if !ok {
		return DamageRecipientCollectionPredicate{}, false
	}
	return DamageRecipientCollectionPredicate{permanents: permanents, players: predicate}, true
}

func (predicate DamageRecipientCollectionPredicate) bindingsKnown(ctx *EffectContext) bool {
	if predicate.targets != nil {
		return ctx != nil && ctx.Game != nil && ctx.Game.GetPlayer(ctx.Controller) != nil && ctx.targetBindings != nil && hasDamageTargetFrame(ctx, predicate.targets)
	}
	return predicate.players.bindingsKnown(ctx) && (predicate.permanents.root == nil || predicate.permanents.root.bindingsKnown(ctx))
}

// Select reads present players and exact current permanent visits without callbacks.
// Supply current layers and a present resolving controller. This query does not
// deal damage, read amounts, use LKI or predict a later game state.
// Target frames must contain the current legal selections from the stack query.
func (predicate DamageRecipientCollectionPredicate) Select(ctx *EffectContext) ([]DamageTargetSelection, bool) {
	if !predicate.bindingsKnown(ctx) {
		return nil, false
	}
	if predicate.targets != nil {
		var out []DamageTargetSelection
		for _, value := range ctx.targetBindings[predicate.targets] {
			if ctx.Game.GetPlayer(value.id) != nil {
				out = append(out, DamageTargetSelection{Player: value.id})
				continue
			}
			permanent := ctx.Game.FindPermanent(value.id)
			if permanent != nil && !permanent.PhasedOut && value.incarnation != uuid.Nil && permanent.incarnationID == value.incarnation {
				out = append(out, DamageTargetSelection{Permanent: PermanentVisit{ID: value.id, Incarnation: value.incarnation}})
			}
		}
		return out, true
	}
	players, known := predicate.players.Select(ctx)
	if !known {
		return nil, false
	}
	var visits []PermanentVisit
	if predicate.permanents.root != nil {
		visits, known = predicate.permanents.selectKnown(ctx)
		if !known {
			return nil, false
		}
	}
	var out []DamageTargetSelection
	for _, player := range players {
		out = append(out, DamageTargetSelection{Player: player})
	}
	for _, visit := range visits {
		out = append(out, DamageTargetSelection{Permanent: visit})
	}
	return out, true
}

func hasDamageTargetFrame(ctx *EffectContext, declaration *targetDeclaration) bool {
	values, ok := ctx.targetBindings[declaration]
	if !ok {
		return false
	}
	seen := make(map[uuid.UUID]bool, len(values))
	for _, value := range values {
		if value.id == uuid.Nil || seen[value.id] {
			return false
		}
		seen[value.id] = true
		if ctx.Game.GetPlayer(value.id) != nil {
			if value.incarnation != uuid.Nil {
				return false
			}
		} else if value.incarnation == uuid.Nil {
			return false
		}
	}
	return true
}
