package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileRequiredSacrificeNumber(payment, continuation Node, ctx scope) (effectFactory, error) {
	if ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(payment, "plural sacrifice payment requires an ordinary resolving body")
	}
	if err := arity(payment, 1); err != nil {
		return nil, err
	}
	cost := payment.Args[0]
	if err := arity(cost, 2); err != nil {
		return nil, err
	}
	amount, err := compileInstanceValue(cost.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	filter, err := compileSacrificeCostFilter(cost.Args[1])
	if err != nil {
		return nil, err
	}
	if continuation.Name != "If" && continuation.Name != "Unless" {
		return nil, problem(continuation, "plural sacrifice payment requires an adjacent If or Unless")
	}
	if err := arity(continuation, 2); err != nil {
		return nil, err
	}
	condition := continuation.Args[0]
	if condition.Kind != Symbol || (condition.Name != "CostWasPaid" && condition.Name != "CostWasntPaid") {
		return nil, unsupported(condition, "payment condition")
	}
	parts, err := list(continuation.Args[1])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(continuation, "payment continuation requires at least one action")
	}
	body, err := compileActionSequence(parts, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		effects := make([]mage.Effect, len(body))
		for i, build := range body {
			effects[i] = build(bindings)
		}
		var paid, unpaid mage.ResolvingEffect
		if (continuation.Name == "If") == (condition.Name == "CostWasPaid") {
			paid = mage.Sequence(effects...)
		} else {
			unpaid = mage.Sequence(effects...)
		}
		return mage.RequiredSacrificeNumberOrElse(mage.You, amount(bindings), filter, paid, unpaid)
	}, nil
}
