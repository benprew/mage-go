package ron

import "github.com/benprew/mage-go/pkg/mage"

func compilePlayerCounterPlacement(n Node, ctx scope) (effectFactory, error) {
	count := 2
	amount := 1
	if n.Name == "NumberCountersOfTypeOnPlayer" {
		count = 3
	}
	if err := arity(n, count); err != nil {
		return nil, err
	}
	if count == 3 {
		value := n.Args[0]
		if value.Name != "Integer" {
			return nil, unsupported(value, "player counter amount")
		}
		if err := arity(value, 1); err != nil {
			return nil, err
		}
		if value.Args[0].Kind != IntegerNode || value.Args[0].Integer < 0 {
			return nil, problem(value, "player counter amount requires a nonnegative integer literal")
		}
		amount = value.Args[0].Integer
	}
	counter := n.Args[count-2]
	if counter.Kind != Symbol || counter.Name != "PoisonCounter" {
		return nil, unsupported(counter, "player counter type")
	}
	player, err := compilePlayerRef(n.Args[count-1], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.AddPoisonFor(player.build(bindings), mage.Fixed(amount))
	}, nil
}
