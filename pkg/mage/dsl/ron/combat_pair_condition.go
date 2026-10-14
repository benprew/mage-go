package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileCombatPairCondition(n Node, ctx scope) (func(instanceBindings) mage.StateCondition, error) {
	if n.Name == "And" || n.Name == "Or" {
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return nil, err
		}
		if len(parts) == 0 {
			return nil, problem(n, "combat condition requires operands")
		}
		plans := make([]func(instanceBindings) mage.StateCondition, len(parts))
		for i, part := range parts {
			plans[i], err = compileCombatPairCondition(part, ctx)
			if err != nil {
				return nil, err
			}
		}
		return func(bindings instanceBindings) mage.StateCondition {
			conditions := make([]mage.StateCondition, len(plans))
			for i, plan := range plans {
				conditions[i] = plan(bindings)
			}
			if n.Name == "And" {
				return mage.AllConditions(conditions...)
			}
			return mage.AnyCondition(conditions...)
		}, nil
	}
	minimum := 0
	filterIndex := 0
	switch n.Name {
	case "NoPermanentsPassFilter":
		if err := arity(n, 2); err != nil {
			return nil, err
		}
	case "NumPermanentsPassFilter":
		if err := arity(n, 3); err != nil {
			return nil, err
		}
		comparison := n.Args[0]
		if comparison.Name != "GreaterThanOrEqualTo" {
			return nil, unsupported(comparison, "combat set count comparison")
		}
		if err := arity(comparison, 1); err != nil {
			return nil, err
		}
		value := comparison.Args[0]
		if value.Name != "Integer" {
			return nil, unsupported(value, "combat set count minimum")
		}
		if err := arity(value, 1); err != nil {
			return nil, err
		}
		if value.Args[0].Kind != IntegerNode || value.Args[0].Integer < 0 {
			return nil, problem(value, "combat set count requires a nonnegative integer")
		}
		minimum = value.Args[0].Integer
		filterIndex = 1
	default:
		return nil, unsupported(n, "combat pair condition")
	}
	filter, err := compileCombatConditionCollection(n.Args[filterIndex], ctx)
	if err != nil {
		return nil, err
	}
	relation := n.Args[filterIndex+1]
	if relation.Name != "IsBlockingAttacker" && relation.Name != "IsBlockedByDefender" {
		return nil, unsupported(relation, "combat condition relation")
	}
	subjects, err := compileContextualCollection(relation, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.StateCondition {
		set := mage.IntersectPermanents(subjects(bindings), filter(bindings))
		if n.Name == "NoPermanentsPassFilter" {
			return mage.PermanentSetIsEmpty(set)
		}
		return mage.PermanentSetCountAtLeast(set, minimum)
	}, nil
}

func compileCombatConditionCollection(n Node, ctx scope) (permanentSetFactory, error) {
	switch n.Name {
	case "Other":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		ref, err := compilePermanentRef(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.PermanentSet {
			return mage.WithoutPermanents(mage.MatchingPermanents(mage.AnyPermanent), mage.SingletonPermanent(ref.build(bindings)))
		}, nil
	case "And", "Or":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return nil, err
		}
		if len(parts) == 0 {
			return nil, problem(n, "combat filter requires operands")
		}
		plans := make([]permanentSetFactory, len(parts))
		for i, part := range parts {
			plans[i], err = compileCombatConditionCollection(part, ctx)
			if err != nil {
				return nil, err
			}
		}
		return func(bindings instanceBindings) mage.PermanentSet {
			sets := make([]mage.PermanentSet, len(plans))
			for i, plan := range plans {
				sets[i] = plan(bindings)
			}
			if n.Name == "And" {
				return mage.IntersectPermanents(sets...)
			}
			return mage.UnionPermanents(sets...)
		}, nil
	default:
		if n.Name == "IsCreatureType" || n.Name == "IsNonCreatureType" {
			if err := arity(n, 1); err != nil {
				return nil, err
			}
			subtype, err := symbol(n.Args[0])
			if err != nil {
				return nil, err
			}
			if err := mage.ValidateSubtypeFamily(mage.SubtypeCreature, subtypeName(subtype)); err != nil {
				return nil, problem(n, err.Error())
			}
		}
		filter, err := compileResolvingPropertyFilter(n)
		if err != nil {
			return nil, err
		}
		return func(instanceBindings) mage.PermanentSet { return mage.MatchingPermanents(filter) }, nil
	}
}
