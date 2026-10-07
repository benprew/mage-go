package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileDepartureDelay(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	duration, err := symbol(n.Args[2])
	if err != nil {
		return nil, err
	}
	if duration != "UntilEndOfTurn" {
		return nil, unsupported(n.Args[2], "delayed departure duration")
	}
	event := n.Args[0]
	if event.Name != "WhenAPermanentDies" && event.Name != "WhenAPermanentLeavesTheBattlefield" {
		return nil, unsupported(event, "delayed departure event")
	}
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	subject := event.Args[0]
	if subject.Name != "SinglePermanent" {
		return nil, unsupported(subject, "delayed departure subject")
	}
	if err := arity(subject, 1); err != nil {
		return nil, err
	}
	ref, err := compilePermanentRef(subject.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	body := n.Args[1]
	if body.Name != "ActionList" {
		return nil, unsupported(body, "delayed departure body")
	}
	if err := arity(body, 1); err != nil {
		return nil, err
	}
	parts, err := list(body.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(body, "delayed departure requires at least one action")
	}
	ctx.delayedStep = true
	effects, err := compileActionSequence(parts, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		values := make([]mage.Effect, len(effects))
		for i, build := range effects {
			values[i] = build(bindings)
		}
		if event.Name == "WhenAPermanentDies" {
			return mage.WhenDiesThisTurn(ref.build(bindings), mage.Sequence(values...))
		}
		return mage.WhenLeavesThisTurn(ref.build(bindings), mage.Sequence(values...))
	}, nil
}
