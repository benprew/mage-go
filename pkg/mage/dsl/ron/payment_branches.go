package ron

import "github.com/benprew/mage-go/pkg/mage"

func compilePaymentBranches(payment, continuation Node, ctx scope) (effectFactory, error) {
	if payment.Name != "MayCost" && payment.Name != "PlayerMayCost" {
		return nil, unsupported(payment, "two-branch payment")
	}
	payer := Node{Kind: Symbol, Name: "You"}
	count := 1
	if payment.Name == "PlayerMayCost" {
		count = 2
	}
	if err := arity(payment, count); err != nil {
		return nil, err
	}
	if payment.Name == "PlayerMayCost" {
		payer = payment.Args[0]
	}
	player, err := compilePlayerRef(payer, ctx)
	if err != nil {
		return nil, err
	}
	cost := payment.Args[len(payment.Args)-1]
	if cost.Name != "PayMana" && cost.Name != "PayLife" {
		return nil, unsupported(cost, "two-branch payment cost")
	}
	costs, hasX, err := compileCosts(cost)
	if err != nil {
		return nil, err
	}
	if hasX {
		return nil, problem(cost, "two-branch payment requires fixed mana")
	}
	if err := arity(continuation, 3); err != nil {
		return nil, err
	}
	condition := continuation.Args[0]
	if condition.Kind != Symbol || (condition.Name != "CostWasPaid" && condition.Name != "CostWasntPaid") {
		return nil, unsupported(condition, "payment condition")
	}
	branches := make([][]effectFactory, 2)
	for i := range branches {
		parts, err := list(continuation.Args[i+1])
		if err != nil {
			return nil, err
		}
		if len(parts) == 0 {
			return nil, problem(continuation.Args[i+1], "payment branch requires at least one action")
		}
		branches[i], err = compileActionSequence(parts, ctx)
		if err != nil {
			return nil, err
		}
	}
	return func(bindings instanceBindings) mage.Effect {
		values := make([]mage.Cost, len(costs))
		for i, build := range costs {
			values[i] = build()
		}
		effects := make([]mage.ResolvingEffect, 2)
		for i, branch := range branches {
			body := make([]mage.Effect, len(branch))
			for j, build := range branch {
				body[j] = build(bindings)
			}
			effects[i] = mage.Sequence(body...)
		}
		if condition.Name == "CostWasntPaid" {
			effects[0], effects[1] = effects[1], effects[0]
		}
		return mage.PayOrElse(player.build(bindings), mage.Costs(values...), effects[0], effects[1])
	}, nil
}
