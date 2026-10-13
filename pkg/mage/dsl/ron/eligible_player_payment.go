package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileEligiblePlayerPayment(payment, continuation Node, ctx scope) (effectFactory, error) {
	if ctx.drawReplacementBody || ctx.enteringSource {
		return nil, problem(payment, "eligible player payment requires an ordinary resolving body")
	}
	if err := arity(payment, 2); err != nil {
		return nil, err
	}
	players := payment.Args[0]
	if players.Kind != Symbol || (players.Name != "Opponent" && players.Name != "AnyPlayer") {
		return nil, unsupported(players, "eligible payment players")
	}
	cost := payment.Args[1]
	if cost.Name != "PayMana" {
		return nil, unsupported(cost, "eligible player payment cost")
	}
	costs, hasX, err := compileCosts(cost)
	if err != nil {
		return nil, err
	}
	if hasX {
		return nil, problem(cost, "eligible player payment requires fixed mana")
	}
	count := 2
	if continuation.Name == "IfElse" {
		count = 3
	} else if continuation.Name != "If" && continuation.Name != "Unless" {
		return nil, problem(continuation, "payment requires an adjacent payment continuation")
	}
	if err := arity(continuation, count); err != nil {
		return nil, err
	}
	condition := continuation.Args[0]
	if condition.Kind != Symbol || (condition.Name != "CostWasPaid" && condition.Name != "CostWasntPaid") {
		return nil, unsupported(condition, "payment condition")
	}
	branches := make([][]effectFactory, count-1)
	ctx.actionPlayer = nil
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
		bodies := make([]mage.ResolvingEffect, 2)
		for i, branch := range branches {
			effects := make([]mage.Effect, len(branch))
			for j, build := range branch {
				effects[j] = build(bindings)
			}
			bodies[i] = mage.Sequence(effects...)
		}
		if (continuation.Name == "Unless") == (condition.Name == "CostWasPaid") {
			bodies[0], bodies[1] = bodies[1], bodies[0]
		}
		eligible := mage.PlayerSet(mage.EachPlayer)
		if players.Name == "Opponent" {
			eligible = mage.OpponentsOf(mage.You)
		}
		return mage.PayAnyOrElse(eligible, mage.Costs(values...), bodies[0], bodies[1])
	}, nil
}
