package ron

import "github.com/benprew/mage-go/pkg/mage"

func drawnReturnInteger(n Node) (int, error) {
	if n.Name != "Integer" || n.Kind != Tuple || len(n.Args) != 1 || n.Args[0].Kind != IntegerNode || n.Args[0].Integer < 0 {
		return 0, problem(n, "requires a fixed nonnegative Integer")
	}
	return n.Args[0].Integer, nil
}

func compileDrawnCardReturn(n Node, ctx scope) ([]effectFactory, error) {
	if !ctx.resolving || ctx.drawReplacementBody {
		return nil, problem(n, "drawn card choice requires an ordinary resolving body")
	}
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	count, err := drawnReturnInteger(n.Args[0])
	if err != nil {
		return nil, err
	}
	life, err := drawnReturnInteger(n.Args[1])
	if err != nil {
		return nil, err
	}
	return []effectFactory{func(bindings instanceBindings) mage.Effect {
		return mage.PayLifeOrReturnDrawnCards(ctx.player.build(bindings), count, life)
	}}, nil
}

func compileOptionalDrawContinuation(payment, continuation Node, ctx scope) (effectFactory, error) {
	if !ctx.resolving || ctx.drawReplacementBody {
		return nil, problem(payment, "optional draws require an ordinary resolving body")
	}
	if err := arity(payment, 1); err != nil {
		return nil, err
	}
	draw := payment.Args[0]
	if err := arity(draw, 1); err != nil {
		return nil, err
	}
	count, err := drawnReturnInteger(draw.Args[0])
	if err != nil {
		return nil, err
	}
	if continuation.Name != "If" {
		return nil, problem(continuation, "optional draws require an If(CostWasPaid, actions) continuation")
	}
	if err := arity(continuation, 2); err != nil {
		return nil, err
	}
	condition := continuation.Args[0]
	if condition.Kind != Symbol || condition.Name != "CostWasPaid" {
		return nil, problem(condition, "optional draws require CostWasPaid")
	}
	parts, err := list(continuation.Args[1])
	if err != nil {
		return nil, err
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
		return mage.MayDrawCardsThen(mage.You, count, mage.Sequence(effects...))
	}, nil
}
