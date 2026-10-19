package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileRandomDiscard(n Node, ctx scope) ([]effectFactory, error) {
	amount := func(instanceBindings) mage.ResolutionValue { return mage.Fixed(1) }
	if n.Name == "DiscardACardAtRandom" {
		if n.Kind != Symbol {
			return nil, problem(n, "DiscardACardAtRandom requires a symbol")
		}
	} else {
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		value := n.Args[0]
		if value.Name == "Integer" {
			if err := arity(value, 1); err != nil {
				return nil, err
			}
			if value.Args[0].Kind != IntegerNode || value.Args[0].Integer < 0 {
				return nil, problem(value, "discard amount must be a nonnegative integer")
			}
		}
		var err error
		amount, err = compileInstanceValue(value, ctx)
		if err != nil {
			return nil, err
		}
	}
	return []effectFactory{func(bindings instanceBindings) mage.Effect {
		return (mage.RandomDiscardInstruction{Subject: ctx.player.build(bindings), Amount: amount(bindings)}).Effect()
	}}, nil
}
