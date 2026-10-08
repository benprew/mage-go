package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileFight(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	first, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	second, err := compilePermanentRef(n.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.Fight(first.build(bindings), second.build(bindings))
	}, nil
}
