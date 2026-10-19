package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileDestructionCollection(n Node, ctx scope) (permanentSetFactory, error) {
	if (n.Name == "And" || n.Name == "Or") && !isDeathCombatCollection(n) && !isLeastPowerCollection(n) {
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return nil, err
		}
		if len(parts) == 0 {
			return nil, problem(n, "collection requires at least one operand")
		}
		plans := make([]permanentSetFactory, len(parts))
		for i, part := range parts {
			plans[i], err = compileDestructionCollection(part, ctx)
			if err != nil {
				return nil, err
			}
		}
		return func(bindings instanceBindings) mage.PermanentSet {
			sets := make([]mage.PermanentSet, len(plans))
			for i, build := range plans {
				sets[i] = build(bindings)
			}
			if n.Name == "And" {
				return mage.IntersectPermanents(sets...)
			}
			return mage.UnionPermanents(sets...)
		}, nil
	}
	build, err := compileContextualCollection(n, ctx)
	if err != nil {
		return nil, err
	}
	if n.Name == "And" || n.Name == "Or" || n.Name == "IsColor" || n.Name == "IsNonColor" {
		if filter, err := compileFilter(n); err == nil {
			if _, ok := mage.InspectPermanentCollection(mage.MatchingPermanents(filter)); ok {
				return func(instanceBindings) mage.PermanentSet { return mage.MatchingPermanents(filter) }, nil
			}
		}
	}
	return build, nil
}
