package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileActionSequence(parts []Node, ctx scope) ([]effectFactory, error) {
	var effects []effectFactory
	for i := 0; i < len(parts); i++ {
		part := parts[i]
		if part.Name == "PlayerMayCost" || part.Name == "MayCost" {
			if i+1 == len(parts) {
				return nil, problem(part, "payment requires an adjacent unpaid continuation")
			}
			effect, err := compileOptionalPayment(part, parts[i+1], ctx)
			if err != nil {
				return nil, err
			}
			effects = append(effects, effect)
			i++
			continue
		}
		compiled, err := compileEffect(part, ctx)
		if err != nil {
			return nil, err
		}
		effects = append(effects, compiled...)
	}
	return effects, nil
}

func compileOptionalPayment(payment, continuation Node, ctx scope) (effectFactory, error) {
	var player playerReferencePlan
	var cost Node
	if payment.Name == "MayCost" {
		if err := arity(payment, 1); err != nil {
			return nil, err
		}
		cost = payment.Args[0]
	} else {
		if err := arity(payment, 2); err != nil {
			return nil, err
		}
		var err error
		player, err = compilePlayerRef(payment.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		cost = payment.Args[1]
	}
	if cost.Name != "PayMana" {
		return nil, unsupported(cost, "optional payment cost")
	}
	costs, _, err := compileCosts(cost)
	if err != nil {
		return nil, err
	}
	if continuation.Name != "Unless" {
		return nil, problem(continuation, "payment requires an adjacent unpaid continuation")
	}
	if err := arity(continuation, 2); err != nil {
		return nil, err
	}
	condition := continuation.Args[0]
	if condition.Kind != Symbol || condition.Name != "CostWasPaid" {
		return nil, unsupported(condition, "payment condition")
	}
	parts, err := list(continuation.Args[1])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(continuation, "unpaid continuation requires at least one action")
	}
	body, err := compileActionSequence(parts, ctx)
	if err != nil {
		return nil, err
	}
	// Keep the payment and its result together so other payments cannot change it.
	return func(bindings instanceBindings) mage.Effect {
		values := make([]mage.Cost, len(costs))
		for i, build := range costs {
			values[i] = build()
		}
		effects := make([]mage.Effect, len(body))
		for i, build := range body {
			effects[i] = build(bindings)
		}
		return mage.UnlessPaid(player.build(bindings), mage.Costs(values...), mage.Sequence(effects...))
	}, nil
}
