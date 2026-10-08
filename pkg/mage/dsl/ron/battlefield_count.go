package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileBattlefieldCount(n Node, ctx scope) (valueFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	subjects, err := compileContextualCollection(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.ResolutionValue {
		return mage.CountPermanents(subjects(bindings))
	}, nil
}
