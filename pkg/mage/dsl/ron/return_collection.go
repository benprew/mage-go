package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileReturnCollection(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	set, err := compileContextualCollection(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.ReturnEachToHand(set(bindings))
	}, nil
}
