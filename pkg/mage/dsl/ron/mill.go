package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileMillCount(n Node, cost bool) (int, error) {
	if err := arity(n, 1); err != nil {
		return 0, err
	}
	value := n.Args[0]
	if value.Name != "Integer" {
		return 0, unsupported(value, "fixed mill amount")
	}
	if err := arity(value, 1); err != nil {
		return 0, err
	}
	if value.Args[0].Kind != IntegerNode || value.Args[0].Integer < 0 {
		return 0, problem(value, "mill amount must be a nonnegative integer")
	}
	amount := value.Args[0].Integer
	if cost && amount == 0 {
		return 0, problem(value, "mill cost requires a positive integer")
	}
	return amount, nil
}

func compileMillAction(n Node, ctx scope) ([]effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	if n.Args[0].Name == "Integer" {
		if _, err := compileMillCount(n, false); err != nil {
			return nil, err
		}
	}
	amount, err := compileInstanceValue(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return []effectFactory{func(bindings instanceBindings) mage.Effect {
		return mage.Mill(ctx.player.build(bindings), amount(bindings))
	}}, nil
}
