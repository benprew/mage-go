package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileSacrificeAllBut(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	subjects, err := compileContextualCollection(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	var keep valueFactory
	amount := n.Args[1]
	switch amount.Name {
	case "Integer":
		if err := arity(amount, 1); err != nil {
			return nil, err
		}
		if amount.Args[0].Kind != IntegerNode || amount.Args[0].Integer < 0 {
			return nil, problem(amount, "kept count must be a nonnegative integer")
		}
		keep = func(instanceBindings) mage.ResolutionValue { return mage.Fixed(amount.Args[0].Integer) }
	case "TheLowestNumberOfPermanentsControlledAmongPlayers":
		if err := arity(amount, 2); err != nil {
			return nil, err
		}
		if amount.Args[1].Kind != Symbol || amount.Args[1].Name != "AnyPlayer" {
			return nil, unsupported(amount.Args[1], "least-count player collection")
		}
		compared, err := compileContextualCollection(amount.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		keep = func(bindings instanceBindings) mage.ResolutionValue {
			return mage.LeastControlledPermanentCount(mage.EachPlayer, compared(bindings))
		}
	default:
		return nil, unsupported(amount, "kept permanent count")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.SacrificeAllButEach(mage.EachPlayer, subjects(bindings), keep(bindings))
	}, nil
}
