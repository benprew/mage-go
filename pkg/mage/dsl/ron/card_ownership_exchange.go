package ron

import "github.com/benprew/mage-go/pkg/mage"

type sourceCardPaymentPlan struct {
	producers int
	used      bool
}

func compileOwnershipExchangeCard(n Node, ctx scope) (func(instanceBindings) mage.CardRef, error) {
	if n.Name == "ThisPermanent" {
		if n.Kind != Symbol {
			return nil, problem(n, "source card reference requires a symbol")
		}
		if ctx.groupedExile != nil && ctx.groupedExile.source >= 0 {
			plan := ctx.groupedExile
			return func(bindings instanceBindings) mage.CardRef {
				return mage.SingleExiledCardFrom(bindings.groupedExiles[plan][plan.source])
			}, nil
		}
		if ctx.sourceCardPayment == nil || ctx.sourceCardPayment.producers != 1 {
			return nil, problem(n, "source card reference requires one source payment or singular source exile")
		}
		ctx.sourceCardPayment.used = true
		return func(bindings instanceBindings) mage.CardRef {
			return mage.FromCard(bindings.sourceCards[ctx.sourceCardPayment])
		}, nil
	}
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
