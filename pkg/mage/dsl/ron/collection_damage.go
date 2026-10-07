package ron

import "github.com/benprew/mage-go/pkg/mage"

func compilePermanentDamageCollection(n Node, ctx scope) (func(instanceBindings) mage.PermanentSet, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	subject := n.Args[0]
	if subject.Kind == Symbol && subject.Name == "Ref_TargetPermanents" {
		ref, err := compilePermanentTargetsRef(subject, ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.PermanentSet { return bindings.permanentSets[ref] }, nil
	}
	if subject.Name == "And" {
		if err := arity(subject, 1); err != nil {
			return nil, err
		}
		parts, err := list(subject.Args[0])
		if err != nil {
			return nil, err
		}
		refIndex := -1
		for i, part := range parts {
			if part.Name == "Ref_TargetPermanents" {
				if refIndex >= 0 {
					return nil, problem(part, "supply one permanent collection reference")
				}
				refIndex = i
			}
		}
		if refIndex >= 0 {
			if len(parts) != 2 {
				return nil, problem(subject, "supply one collection reference and one filter")
			}
			ref, err := compilePermanentTargetsRef(parts[refIndex], ctx)
			if err != nil {
				return nil, err
			}
			filter, err := compileResolvingPropertyFilter(parts[1-refIndex])
			if err != nil {
				return nil, err
			}
			return func(bindings instanceBindings) mage.PermanentSet {
				return mage.FilterPermanentSet(bindings.permanentSets[ref], filter)
			}, nil
		}
	}
	filter, err := compileLiveDamageCollectionFilter(subject)
	if err != nil {
		return nil, err
	}
	return func(instanceBindings) mage.PermanentSet { return mage.MatchingPermanents(filter) }, nil
}

func compileLiveDamageCollectionFilter(n Node) (mage.PermanentFilter, error) {
	if n.Name != "And" && n.Name != "Or" {
		return compileFilter(n)
	}
	if err := arity(n, 1); err != nil {
		return mage.PermanentFilter{}, err
	}
	parts, err := list(n.Args[0])
	if err != nil {
		return mage.PermanentFilter{}, err
	}
	if len(parts) == 0 {
		return mage.PermanentFilter{}, problem(n, "damage collection requires a nonempty filter")
	}
	filters := make([]mage.PermanentFilter, len(parts))
	for i, part := range parts {
		filters[i], err = compileLiveDamageCollectionFilter(part)
		if err != nil {
			return mage.PermanentFilter{}, err
		}
	}
	if n.Name == "And" {
		return mage.And(filters...), nil
	}
	return mage.Or(filters...), nil
}
