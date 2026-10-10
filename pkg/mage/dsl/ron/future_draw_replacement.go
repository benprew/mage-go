package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileFutureDrawReplacement(n Node, ctx scope) (effectFactory, error) {
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
	body, err := compileActionSequence(parts, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		effects := make([]mage.Effect, len(body))
		for i, build := range body {
			effects[i] = build(bindings)
		}
		return mage.ReplaceNextDrawThisTurn(player.build(bindings), mage.Sequence(effects...))
	}, nil
}
