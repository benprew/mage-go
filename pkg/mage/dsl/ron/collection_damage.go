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
					return compileContextualCollection(subject, ctx)
				}
				refIndex = i
			}
		}
		if refIndex >= 0 {
			if len(parts) != 2 {
				return compileContextualCollection(subject, ctx)
			}
			ref, err := compilePermanentTargetsRef(parts[refIndex], ctx)
			if err != nil {
				return nil, err
			}
			filter, err := compileResolvingPropertyFilter(parts[1-refIndex])
			if err != nil {
				return compileContextualCollection(subject, ctx)
			}
			return func(bindings instanceBindings) mage.PermanentSet {
				return mage.FilterPermanentSet(bindings.permanentSets[ref], filter)
			}, nil
		}
	}
	return compileContextualCollection(subject, ctx)
}
