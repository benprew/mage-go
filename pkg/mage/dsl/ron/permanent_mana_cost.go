package ron

import "github.com/benprew/mage-go/pkg/mage"

func compilePermanentManaCost(n Node, ctx scope) ([]boundCostFactory, bool, error) {
	if err := arity(n, 1); err != nil {
		return nil, false, err
	}
	subject, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, false, err
	}
	if subject.kind != permanentSource && subject.kind != permanentHost {
		return nil, false, unsupported(n.Args[0], "activation mana cost permanent reference")
	}
	return []boundCostFactory{func(bindings instanceBindings) mage.Cost {
		return mage.PermanentManaCostPayment(subject.build(bindings))
	}}, false, nil
}
