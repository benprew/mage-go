package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileDelayedStep(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	event := n.Args[0]
	var pattern mage.DelayedStepPattern
	switch event.Name {
	case "AtTheBeginningOfTheNextEndStep", "AtTheNextEndOfCombat":
		if _, err := symbol(event); err != nil {
			return nil, err
		}
		pattern = mage.EachEndStep
		if event.Name == "AtTheNextEndOfCombat" {
			pattern = mage.EachEndOfCombat
		}
	case "AtTheBeginningOfPlayersNextUpkeep":
		if err := arity(event, 1); err != nil {
			return nil, err
		}
		player, err := symbol(event.Args[0])
		if err != nil {
			return nil, err
		}
		if player != "You" {
			return nil, unsupported(event.Args[0], "delayed upkeep player")
		}
		pattern = mage.YourUpkeep
	default:
		return nil, unsupported(event, "delayed step event")
	}
	body := n.Args[1]
	if body.Name != "ActionList" {
		return nil, unsupported(body, "delayed step body")
	}
	if err := arity(body, 1); err != nil {
		return nil, err
	}
	parts, err := list(body.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(body, "delayed step requires at least one action")
	}
	ctx.delayedStep = true
	ctx.drawReplacementBody = false
	effects, err := compileActionSequence(parts, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		values := make([]mage.Effect, len(effects))
		for i, build := range effects {
			values[i] = build(bindings)
		}
		return mage.AtNext(pattern, mage.Sequence(values...))
	}, nil
}
