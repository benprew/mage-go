package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileOwnershipExchangeCard(n Node, ctx scope) (func(instanceBindings) mage.CardRef, error) {
	if n.Name == "TheCardRevealedFromHandThisWay" {
		return compileRevealedHandCardRef(n, ctx)
	}
	ref, err := compileCardRef(n, ctx)
	if err != nil {
		return nil, err
	}
	return ref.build, nil
}

func compileCardOwnershipExchange(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	first, err := compileOwnershipExchangeCard(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	second, err := compileOwnershipExchangeCard(n.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.ExchangeCardOwnership(first(bindings), second(bindings))
	}, nil
}
