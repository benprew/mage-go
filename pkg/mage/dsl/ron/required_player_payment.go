package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileRequiredPlayerPayment(payment, continuation Node, ctx scope) (effectFactory, error) {
	if err := arity(payment, 2); err != nil {
		return nil, err
	}
	player, err := compilePlayerRef(payment.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	cost := payment.Args[1]
	if cost.Kind != Symbol || cost.Name != "DiscardACard" {
		return nil, unsupported(cost, "required player payment cost")
	}
	count := 2
	switch continuation.Name {
	case "If", "Unless":
	case "IfElse":
		count = 3
	default:
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
		bodies := make([]mage.ResolvingEffect, len(branches))
		for i, branch := range branches {
			effects := make([]mage.Effect, len(branch))
			for j, build := range branch {
				effects[j] = build(bindings)
			}
			bodies[i] = mage.Sequence(effects...)
		}
		if ctx.drawReplacementBody {
			var discarded, notDiscarded mage.ResolvingEffect
			discarded = bodies[0]
			if len(bodies) == 2 {
				notDiscarded = bodies[1]
			}
			if (condition.Name == "CostWasntPaid") != (continuation.Name == "Unless") {
				discarded, notDiscarded = notDiscarded, discarded
			}
			return mage.DiscardOneOrElse(player.build(bindings), discarded, notDiscarded)
		}
		costs := mage.Costs(mage.DiscardCost(1))
		if len(bodies) == 1 {
			if (condition.Name == "CostWasntPaid") != (continuation.Name == "Unless") {
				return mage.UnlessRequiredPaid(player.build(bindings), costs, bodies[0])
			}
			return mage.IfRequiredPaid(player.build(bindings), costs, bodies[0])
		}
		if condition.Name == "CostWasntPaid" {
			bodies[0], bodies[1] = bodies[1], bodies[0]
		}
		return mage.PayRequiredOrElse(player.build(bindings), costs, bodies[0], bodies[1])
	}, nil
}
