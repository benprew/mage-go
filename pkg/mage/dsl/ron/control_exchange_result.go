package ron

import (
	"maps"
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
)

type controlExchangePlan struct{ label string }

func hasControlExchangeRead(nodes []Node) bool {
	return slices.ContainsFunc(nodes, func(n Node) bool {
		return n.Name == "PermanentsChangedControlThisWay" || hasControlExchangeRead(n.Args)
	})
}

func compileControlExchangeProducer(n Node, remaining []Node, ctx scope) (effectFactory, error) {
	if ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "exchange outcomes require an ordinary resolving body")
	}
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	first, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	second, err := compilePermanentRef(n.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	plan := &controlExchangePlan{label: "exchange at " + n.Position.String()}
	ctx.controlExchange = plan
	body, err := compileActionSequence(remaining, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		bindings.controlExchanges = maps.Clone(bindings.controlExchanges)
		if bindings.controlExchanges == nil {
			bindings.controlExchanges = make(map[*controlExchangePlan]*mage.ControlExchangeResultBinding)
		}
		result := mage.BindControlExchangeResult(plan.label)
		bindings.controlExchanges[plan] = result
		effects := make([]mage.Effect, 0, 1+len(body))
		effects = append(effects, mage.ExchangePermanentControlInto(first.build(bindings), second.build(bindings), result))
		for _, build := range body {
			effects = append(effects, build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}

func compileControlExchangeBranch(n Node, ctx scope) (effectFactory, error) {
	if ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "exchange outcomes require an ordinary resolving body")
	}
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if n.Args[0].Kind != Symbol || ctx.controlExchange == nil {
		return nil, problem(n.Args[0], "unbound or malformed exchange outcome")
	}
	parts, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(n.Args[1], "exchange outcome condition requires actions")
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
		return mage.IfControlExchanged(bindings.controlExchanges[ctx.controlExchange], mage.Sequence(effects...))
	}, nil
}
