package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileSourceCounterCap(n, operation Node) (optionFactory, error) {
	if n.Args[0].Kind != Symbol || n.Args[0].Name != "ThisPermanent" {
		return nil, unsupported(n.Args[0], "counter cap source")
	}
	if err := arity(operation, 2); err != nil {
		return nil, err
	}
	limit := operation.Args[0]
	if limit.Name != "Integer" || limit.Kind != Tuple || len(limit.Args) != 1 || limit.Args[0].Kind != IntegerNode || limit.Args[0].Integer < 0 {
		return nil, problem(limit, "counter cap requires a nonnegative integer literal")
	}
	counter, err := compilePermanentCounterType(operation.Args[1])
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		return mage.WithStaticAbility(mage.SourceCounterCap(counter, limit.Args[0].Integer))
	}, nil
}
