package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileDiscardAllBut(n Node) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	amount := n.Args[0]
	var keep valueFactory
	switch amount.Name {
	case "Integer":
		if err := arity(amount, 1); err != nil {
			return nil, err
		}
		if amount.Args[0].Kind != IntegerNode || amount.Args[0].Integer < 0 {
			return nil, problem(amount, "kept hand count must be a nonnegative integer")
		}
		keep = func(instanceBindings) mage.ResolutionValue { return mage.Fixed(amount.Args[0].Integer) }
	case "TheLowestNumberOfCardsInHandAmongPlayers":
		if err := arity(amount, 1); err != nil {
			return nil, err
		}
		if amount.Args[0].Kind != Symbol || amount.Args[0].Name != "AnyPlayer" {
			return nil, unsupported(amount.Args[0], "least-hand player collection")
		}
		keep = func(instanceBindings) mage.ResolutionValue { return mage.LeastHandSize(mage.EachPlayer) }
	default:
		return nil, unsupported(amount, "kept hand count")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.DiscardAllButEach(mage.EachPlayer, keep(bindings))
	}, nil
}
