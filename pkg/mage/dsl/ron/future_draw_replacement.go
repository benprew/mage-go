package ron

import "github.com/benprew/mage-go/pkg/mage"

type futureDrawReplacementPlan struct {
	player playerReferencePlan
	body   []effectFactory
}

func (p futureDrawReplacementPlan) build(bindings instanceBindings) mage.Effect {
	effects := make([]mage.Effect, len(p.body))
	for i, build := range p.body {
		effects[i] = build(bindings)
	}
	return mage.ReplaceNextDrawThisTurn(p.player.build(bindings), mage.Sequence(effects...))
}

func compileFutureDrawReplacement(n Node, ctx scope) (effectFactory, error) {
	plan, err := compileFutureDrawReplacementPlan(n, ctx)
	if err != nil {
		return nil, err
	}
	return plan.build, nil
}

func compileFutureDrawReplacementPlan(n Node, ctx scope) (*futureDrawReplacementPlan, error) {
	if n.Name != "CreateFutureReplaceWouldDraw" {
		return nil, unsupported(n, "future draw replacement")
	}
	if !ctx.resolving || ctx.enteringSource {
		return nil, problem(n, "future draw replacement requires a resolving body")
	}
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	event := n.Args[0]
	if event.Name != "NextTimePlayerWouldDrawACardThisTurn" {
		return nil, unsupported(event, "future draw replacement event")
	}
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	player, err := compilePlayerRef(event.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	parts, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(n.Args[1], "supply a future draw replacement body")
	}
	ctx.drawReplacementBody = true
	body, err := compileActionSequence(parts, ctx)
	if err != nil {
		return nil, err
	}
	return &futureDrawReplacementPlan{player: player, body: body}, nil
}
