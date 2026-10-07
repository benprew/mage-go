package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
)

func compileDestructionPayment(payment, continuation Node, ctx scope) (effectFactory, error) {
	if err := arity(payment, 1); err != nil {
		return nil, err
	}
	instruction := payment.Args[0]
	if err := arity(instruction, 1); err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(instruction.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if continuation.Name != "If" && continuation.Name != "Unless" {
		return nil, problem(continuation, "destruction requires an adjacent result continuation")
	}
	if err := arity(continuation, 2); err != nil {
		return nil, err
	}
	condition := continuation.Args[0]
	if condition.Kind != Symbol || (condition.Name != "CostWasPaid" && condition.Name != "CostWasntPaid") {
		return nil, unsupported(condition, "destruction result condition")
	}
	parts, err := list(continuation.Args[1])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(continuation, "destruction continuation requires at least one action")
	}
	body, err := compileActionSequence(parts, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		result := mage.BindNumber("destruction at " + payment.Position.String())
		effects := make([]mage.Effect, len(body))
		for i, build := range body {
			effects[i] = build(bindings)
		}
		value := mage.FromNumber(result)
		if (continuation.Name == "If") != (condition.Name == "CostWasPaid") {
			value = mage.SubtractValues(mage.Fixed(1), value)
		}
		return mage.Sequence(mage.DestroyWithResult(subject.build(bindings), mage.DestroyedThisWayInto(result)), mage.IfPositive(value, mage.Sequence(effects...)))
	}, nil
}
