package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileCounterPlacementHistory(n Node, ctx scope) (permanentSetFactory, error) {
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "counter placement history requires an ordinary resolving collection")
	}
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	source, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	counter, err := compilePermanentCounterType(n.Args[1])
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.PermanentSet {
		return mage.PermanentsGivenCountersBy(source.build(bindings), counter, mage.MatchingPermanents(mage.AnyPermanent))
	}, nil
}
