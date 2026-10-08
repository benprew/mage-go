package ron

import "github.com/benprew/mage-go/pkg/mage"

type permanentSetFactory func(instanceBindings) mage.PermanentSet

func compileCombatCollection(n Node, ctx scope) (permanentSetFactory, error) {
	if n.Name != "And" {
		return nil, unsupported(n, "combat relation collection")
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	parts, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) != 2 {
		return nil, problem(n, "combat collection requires a creature type and one relation")
	}
	creature, relation := parts[0], parts[1]
	if relation.Name == "IsCardtype" {
		creature, relation = relation, creature
	}
	if creature.Name != "IsCardtype" {
		return nil, unsupported(creature, "combat collection creature type")
	}
	if err := arity(creature, 1); err != nil {
		return nil, err
	}
	if creature.Args[0].Kind != Symbol || creature.Args[0].Name != "Creature" {
		return nil, unsupported(creature, "combat collection creature type")
	}
	if relation.Name != "IsBlockingAttacker" && relation.Name != "IsBlockedByDefender" {
		return nil, unsupported(relation, "combat collection relation")
	}
	if err := arity(relation, 1); err != nil {
		return nil, err
	}
	ref, err := compilePermanentRef(relation.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.PermanentSet {
		if relation.Name == "IsBlockingAttacker" {
			return mage.BlockersOf(ref.build(bindings))
		}
		return mage.BlockedAttackersOf(ref.build(bindings))
	}, nil
}

func compileCombatCollectionPrevention(n Node, ctx scope) (effectFactory, error) {
	event := n.Args[0]
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	collection := event.Args[0]
	if collection.Name != "Or" {
		return nil, unsupported(collection, "combat prevention collection union")
	}
	if err := arity(collection, 1); err != nil {
		return nil, err
	}
	parts, err := list(collection.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) != 2 {
		return nil, problem(collection, "combat prevention requires one permanent and one relation collection")
	}
	single, related := parts[0], parts[1]
	if related.Name == "SinglePermanent" {
		single, related = related, single
	}
	if single.Name != "SinglePermanent" {
		return nil, unsupported(single, "combat prevention single permanent")
	}
	if err := arity(single, 1); err != nil {
		return nil, err
	}
	ref, err := compilePermanentRef(single.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	set, err := compileCombatCollection(related, ctx)
	if err != nil {
		return nil, err
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 || operations[0].Kind != Symbol || operations[0].Name != "PreventThatDamage" {
		return nil, problem(n.Args[1], "combat prevention requires the symbol PreventThatDamage")
	}
	duration := n.Args[2]
	if duration.Kind != Symbol || (duration.Name != "UntilEndOfTurn" && duration.Name != "UntilEndOfCombat") {
		return nil, unsupported(duration, "combat collection prevention duration")
	}
	return func(bindings instanceBindings) mage.Effect {
		subjects := mage.UnionPermanents(mage.SingletonPermanent(ref.build(bindings)), set(bindings))
		if duration.Name == "UntilEndOfTurn" {
			return mage.PreventCombatDamageByEachThisTurn(subjects)
		}
		return mage.PreventCombatDamageByEach(subjects)
	}, nil
}
