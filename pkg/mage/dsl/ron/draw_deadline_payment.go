package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileDrawDeadlinePayment(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 4); err != nil {
		return nil, err
	}
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "draw deadline requires an ordinary resolving body")
	}
	event := n.Args[0]
	if event.Name != "AtTheBeginningOfPlayersNextDrawStep" {
		return nil, unsupported(event, "payment deadline event")
	}
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	if !sameDeclaration(event.Args[0], n.Args[2]) {
		return nil, problem(n.Args[2], "payer must match the draw deadline player")
	}
	player, err := compilePlayerRef(n.Args[2], ctx)
	if err != nil {
		return nil, err
	}
	text, err := manaCost(n.Args[3])
	if err != nil {
		return nil, err
	}
	mana := core.ParseManaCost(text)
	if mana.HasX {
		return nil, problem(n.Args[3], "draw deadline requires fixed mana")
	}
	body := n.Args[1]
	if body.Name != "ActionList" {
		return nil, unsupported(body, "draw deadline body")
	}
	if err := arity(body, 1); err != nil {
		return nil, err
	}
	parts, err := list(body.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(body, "draw deadline requires at least one action")
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
		return mage.UnlessManaPaidBeforeNextDraw(player.build(bindings), mana, mage.Sequence(values...))
	}, nil
}
