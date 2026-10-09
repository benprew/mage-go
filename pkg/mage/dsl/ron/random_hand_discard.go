package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileRandomHandDiscard(n Node, ctx scope) ([]effectFactory, error) {
	if n.Name == "RevealHandAndDiscardACardOfTypeAtRandom" {
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		filter, err := compileHandCardFilter(n.Args[0])
		if err != nil {
			return nil, err
		}
		return []effectFactory{func(bindings instanceBindings) mage.Effect {
			return mage.RevealHandAndDiscardRandomMatching(ctx.player.build(bindings), filter)
		}}, nil
	}
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	value := n.Args[0]
	if value.Name == "Integer" {
		if err := arity(value, 1); err != nil {
			return nil, err
		}
		if value.Args[0].Kind != IntegerNode || value.Args[0].Integer < 0 {
			return nil, problem(value, "reveal amount must be a nonnegative integer")
		}
	}
	amount, err := compileInstanceValue(value, ctx)
	if err != nil {
		return nil, err
	}
	matcher, err := compileResolvingCardNameFilter(n.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	return []effectFactory{func(bindings instanceBindings) mage.Effect {
		return mage.RevealRandomHandAndDiscardMatchingName(ctx.player.build(bindings), amount(bindings), matcher(bindings))
	}}, nil
}
