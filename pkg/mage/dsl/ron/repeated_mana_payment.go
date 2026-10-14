package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileRepeatedManaPayment(payment, continuation Node, ctx scope) (effectFactory, error) {
	if ctx.enteringSource || ctx.drawReplacementBody || (payment.Name != "MayCost" && payment.Name != "PlayerMayCost") {
		return nil, unsupported(payment, "repeated resolving mana payment")
	}
	payer := Node{Kind: Symbol, Name: "You"}
	arguments := 1
	if payment.Name == "PlayerMayCost" {
		arguments = 2
	}
	if err := arity(payment, arguments); err != nil {
		return nil, err
	}
	if arguments == 2 {
		payer = payment.Args[0]
	}
	player, err := compilePlayerRef(payer, ctx)
	if err != nil {
		return nil, err
	}
	cost := payment.Args[arguments-1]
	if err := arity(cost, 2); err != nil {
		return nil, err
	}
	text, err := manaCost(cost.Args[0])
	if err != nil {
		return nil, err
	}
	mana := core.ParseManaCost(text)
	if mana.HasX {
		return nil, problem(cost, "repeated mana requires a fixed mana list")
	}
	amount := cost.Args[1]
	count, err := compileInstanceValue(amount, ctx)
	if err != nil {
		return nil, err
	}
	if amount.Name == "Integer" && amount.Args[0].Integer < 0 {
		return nil, problem(amount, "repeated mana requires a nonnegative integer")
	}
	branches := 1
	switch continuation.Name {
	case "If", "Unless":
	case "IfElse":
		branches = 2
	default:
		return nil, problem(continuation, "payment requires an adjacent payment continuation")
	}
	if err := arity(continuation, branches+1); err != nil {
		return nil, err
	}
	condition := continuation.Args[0]
	if condition.Kind != Symbol || (condition.Name != "CostWasPaid" && condition.Name != "CostWasntPaid") {
		return nil, unsupported(condition, "payment condition")
	}
	bodies := make([][]effectFactory, branches)
	for i := range bodies {
		parts, err := list(continuation.Args[i+1])
		if err != nil {
			return nil, err
		}
		if len(parts) == 0 {
			return nil, problem(continuation.Args[i+1], "payment branch requires at least one action")
		}
		bodies[i], err = compileActionSequence(parts, ctx)
		if err != nil {
			return nil, err
		}
	}
	return func(bindings instanceBindings) mage.Effect {
		effects := make([]mage.ResolvingEffect, len(bodies))
		for i, body := range bodies {
			values := make([]mage.Effect, len(body))
			for j, build := range body {
				values[j] = build(bindings)
			}
			effects[i] = mage.Sequence(values...)
		}
		if len(effects) == 2 {
			if condition.Name == "CostWasntPaid" {
				effects[0], effects[1] = effects[1], effects[0]
			}
			return mage.PayRepeatedManaOrElse(player.build(bindings), mana, count(bindings), effects[0], effects[1])
		}
		if (continuation.Name == "If") == (condition.Name == "CostWasPaid") {
			return mage.IfRepeatedManaPaid(player.build(bindings), mana, count(bindings), effects[0])
		}
		return mage.UnlessRepeatedManaPaid(player.build(bindings), mana, count(bindings), effects[0])
	}, nil
}
