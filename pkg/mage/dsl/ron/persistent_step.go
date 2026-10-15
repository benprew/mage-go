package ron

import "github.com/benprew/mage-go/pkg/mage"

func compilePersistentStep(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "persistent upkeep requires an ordinary resolving body")
	}
	event := n.Args[0]
	if event.Name != "AtTheBeginningOfAPlayersUpkeep" {
		return nil, unsupported(event, "persistent step event")
	}
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	player := event.Args[0]
	if player.Name != "SinglePlayer" {
		return nil, unsupported(player, "persistent upkeep player")
	}
	if err := arity(player, 1); err != nil {
		return nil, err
	}
	name, err := symbol(player.Args[0])
	if err != nil {
		return nil, err
	}
	if name != "You" {
		return nil, unsupported(player.Args[0], "persistent upkeep player")
	}
	body := n.Args[1]
	if body.Name != "ActionList" {
		return nil, unsupported(body, "persistent upkeep body")
	}
	if err := arity(body, 1); err != nil {
		return nil, err
	}
	parts, err := list(body.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(body, "persistent upkeep requires at least one action")
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
		return mage.AtEach(mage.YourUpkeep, mage.Sequence(values...))
	}, nil
}
