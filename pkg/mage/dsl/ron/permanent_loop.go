package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

type permanentLoopPlan struct {
	name        string
	deathResult bool
}

func compilePermanentLoop(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "permanent iteration requires an ordinary resolving body")
	}
	subjects, err := compileContextualCollection(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	parts, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	plan := &permanentLoopPlan{name: "permanent loop " + n.Position.String()}
	ctx.permanentLoop = plan
	body, err := compileActionSequence(parts, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		collection := subjects(bindings)
		binding := mage.BindPermanent(plan.name)
		bindings.permanentLoops = maps.Clone(bindings.permanentLoops)
		if bindings.permanentLoops == nil {
			bindings.permanentLoops = make(map[*permanentLoopPlan]*mage.PermanentBinding)
		}
		bindings.permanentLoops[plan] = binding
		effects := make([]mage.Effect, len(body))
		for i, build := range body {
			effects[i] = build(bindings)
		}
		return mage.ForEachInTurnOrder(collection, binding, mage.Sequence(effects...))
	}, nil
}
