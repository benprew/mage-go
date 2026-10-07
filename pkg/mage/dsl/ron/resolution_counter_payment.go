package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileResolutionCounterPayment(payment, continuation Node, ctx scope) (effectFactory, error) {
	count := 1
	payer := Node{Kind: Symbol, Name: "You"}
	if payment.Name == "PlayerMayCost" {
		count = 2
	}
	if err := arity(payment, count); err != nil {
		return nil, err
	}
	if count == 2 {
		payer = payment.Args[0]
	}
	player, err := compilePlayerRef(payer, ctx)
	if err != nil {
		return nil, err
	}
	removal := payment.Args[count-1]
	if err := arity(removal, 1); err != nil {
		return nil, err
	}
	parts, err := list(removal.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) != 1 {
		return nil, problem(removal, "counter payment requires one source instruction")
	}
	instruction := parts[0]
	chosen := instruction.Name == "AnyNumberOfCountersOfTypeFromPermanent"
	var plan *counterPaymentPlan
	var counter core.CounterType
	amount := 1
	if chosen {
		if payment.Name == "MustCost" {
			return nil, unsupported(payment, "mandatory chosen counter payment")
		}
		plan, err = compileCounterPaymentPlan(removal)
	} else {
		_, err = compileCounterRemovalCost(removal)
		if err == nil {
			index := 0
			if instruction.Name == "NumberCountersOfTypeFromPermanent" {
				index = 1
				amount = instruction.Args[0].Args[0].Integer
			}
			counter, err = compilePermanentCounterType(instruction.Args[index])
		}
	}
	if err != nil {
		return nil, err
	}
	if !ctx.permanent {
		return nil, problem(removal, "counter payment requires a permanent source")
	}
	if continuation.Name != "If" && continuation.Name != "Unless" {
		return nil, unsupported(continuation, "counter payment continuation")
	}
	if err := arity(continuation, 2); err != nil {
		return nil, err
	}
	condition := continuation.Args[0]
	if condition.Kind != Symbol || (condition.Name != "CostWasPaid" && condition.Name != "CostWasntPaid") {
		return nil, unsupported(condition, "payment condition")
	}
	onPaid := (continuation.Name == "If") == (condition.Name == "CostWasPaid")
	branchCtx := ctx
	// A payment result belongs only to its own paid continuation.
	branchCtx.counterPayment = nil
	if chosen && onPaid {
		branchCtx.counterPayment = plan
	}
	actions, err := list(continuation.Args[1])
	if err != nil {
		return nil, err
	}
	if len(actions) == 0 {
		return nil, problem(continuation, "payment continuation requires at least one action")
	}
	body, err := compileActionSequence(actions, branchCtx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		var result *mage.NumberBinding
		if chosen {
			result = mage.BindNumber("resolution counters at " + payment.Position.String())
			bindings.counters = maps.Clone(bindings.counters)
			if bindings.counters == nil {
				bindings.counters = make(map[*counterPaymentPlan]*mage.NumberBinding)
			}
			bindings.counters[plan] = result
		}
		effects := make([]mage.Effect, len(body))
		for i, build := range body {
			effects[i] = build(bindings)
		}
		var paid = mage.Sequence(effects...)
		var unpaid mage.ResolvingEffect
		if !onPaid {
			paid, unpaid = unpaid, paid
		}
		if chosen {
			return mage.PayChosenCountersOrElse(player.build(bindings), mage.This, plan.counter, result, paid, unpaid)
		}
		return mage.PayCountersOrElse(player.build(bindings), mage.This, counter, amount, payment.Name != "MustCost", paid, unpaid)
	}, nil
}
