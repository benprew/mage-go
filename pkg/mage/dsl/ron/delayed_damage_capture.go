package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

func compileDelayedDamageCapture(n Node, ctx scope) (effectFactory, error) {
	event := n.Args[0]
	if err := arity(event, 2); err != nil {
		return nil, err
	}
	from, err := compilePreventionSourceFilter(event.Args[0])
	if err != nil {
		return nil, err
	}
	subject := event.Args[1]
	if subject.Name != "SinglePermanent" {
		return nil, unsupported(subject, "delayed damage recipient")
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
		return nil, unsupported(body, "delayed damage body")
	}
	if err := arity(body, 1); err != nil {
		return nil, err
	}
	parts, err := list(body.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(body, "delayed damage requires at least one action")
	}
	amount := &damageAmountPlan{name: "Trigger_ThatMuch"}
	ctx.damageAmount = amount
	ctx.delayedStep = true
	ctx.drawReplacementBody = false
	effects, err := compileActionSequence(parts, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		number := mage.BindNumber(amount.name)
		child := bindings
		child.damageAmounts = maps.Clone(bindings.damageAmounts)
		if child.damageAmounts == nil {
			child.damageAmounts = make(map[*damageAmountPlan]*mage.NumberBinding)
		}
		child.damageAmounts[amount] = number
		values := make([]mage.Effect, len(effects))
		for i, build := range effects {
			values[i] = build(child)
		}
		return mage.WhenReceivesDamageThisTurn(ref.build(bindings), from, number, mage.Sequence(values...))
	}, nil
}
