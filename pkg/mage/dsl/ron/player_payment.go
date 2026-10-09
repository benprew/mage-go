package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compilePlayerPayment(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	player, err := compilePlayerRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if n.Args[2].Kind != Symbol || n.Args[2].Name != "UntilEndOfTurn" {
		return nil, unsupported(n.Args[2], "player payment duration")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) == 0 {
		return nil, problem(n.Args[1], "player effect requires at least one operation")
	}
	var payments []effectFactory
	for _, operation := range operations {
		if operation.Name != "MayPayManaAnyTimeTheyCouldCastAnInstant" {
			return nil, unsupported(operation, "temporary player operation")
		}
		if err := arity(operation, 2); err != nil {
			return nil, err
		}
		text, err := manaCost(operation.Args[0])
		if err != nil {
			return nil, err
		}
		mana := core.ParseManaCost(text)
		if mana.HasX {
			return nil, problem(operation.Args[0], "player payment requires fixed mana")
		}
		parts, err := list(operation.Args[1])
		if err != nil {
			return nil, err
		}
		if len(parts) == 0 {
			return nil, problem(operation.Args[1], "player payment requires at least one action")
		}
		body, err := compileActionSequence(parts, ctx)
		if err != nil {
			return nil, err
		}
		payments = append(payments, func(bindings instanceBindings) mage.Effect {
			effects := make([]mage.Effect, len(body))
			for i, build := range body {
				effects[i] = build(bindings)
			}
			return mage.PlayerManaPaymentThisTurn(player.build(bindings), mana, mage.Sequence(effects...))
		})
	}
	return func(bindings instanceBindings) mage.Effect {
		effects := make([]mage.Effect, len(payments))
		for i, build := range payments {
			effects[i] = build(bindings)
		}
		return mage.Sequence(effects...)
	}, nil
}
