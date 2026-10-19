package ron

import "github.com/benprew/mage-go/pkg/mage"

type discardFactPlan struct {
	producers int
	single    bool
	filters   []mage.CardFilter
}

func compileDiscardFactBranch(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if ctx.discardFacts == nil || !ctx.activation || !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "discard fact requires an activated ability discard cost")
	}
	if ctx.discardFacts.producers != 1 || !ctx.discardFacts.single {
		return nil, problem(n, "discard fact requires exactly one single-card discard cost")
	}
	condition := n.Args[0]
	if err := arity(condition, 1); err != nil {
		return nil, err
	}
	filter, err := compileHandCardFilter(condition.Args[0])
	if err != nil {
		return nil, err
	}
	parts, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	index := len(ctx.discardFacts.filters)
	ctx.discardFacts.filters = append(ctx.discardFacts.filters, filter)
	effects, err := compileActionSequence(parts, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		body := make([]mage.Effect, len(effects))
		for i, build := range effects {
			body[i] = build(bindings)
		}
		return mage.IfPositive(mage.FromNumber(bindings.discardFacts[ctx.discardFacts][index]), mage.Sequence(body...))
	}, nil
}
