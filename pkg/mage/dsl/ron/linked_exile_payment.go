package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

type linkedExilePaymentPlan struct{ position string }

func compileLinkedExilePayment(payment, continuation Node, ctx scope) (effectFactory, error) {
	if payment.Name != "MayCost" {
		return nil, unsupported(payment, "linked exile payment")
	}
	if err := arity(payment, 1); err != nil {
		return nil, err
	}
	if !ctx.permanent || !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(payment, "linked exile payment requires an ordinary permanent-source resolving body")
	}
	cost := payment.Args[0]
	if err := arity(cost, 2); err != nil {
		return nil, err
	}
	objects, err := list(cost.Args[0])
	if err != nil {
		return nil, err
	}
	modifiers, err := list(cost.Args[1])
	if err != nil {
		return nil, err
	}
	if len(objects) != 1 || len(modifiers) != 0 {
		return nil, problem(cost, "linked exile payment requires one selection and no modifiers")
	}
	selection := objects[0]
	if selection.Name != "AnyNumberOfPermanents" {
		return nil, unsupported(selection, "linked exile selection")
	}
	if err := arity(selection, 1); err != nil {
		return nil, err
	}
	subject := selection.Args[0]
	if subject.Name != "TheCreatedTokens" || subject.Kind != Symbol {
		return nil, problem(subject, "linked exile payment requires TheCreatedTokens")
	}
	set, err := compileContextualCollection(subject, ctx)
	if err != nil {
		return nil, err
	}
	if continuation.Name != "If" && continuation.Name != "Unless" {
		return nil, unsupported(continuation, "linked exile payment continuation")
	}
	if err := arity(continuation, 2); err != nil {
		return nil, err
	}
	condition := continuation.Args[0]
	if condition.Kind != Symbol || (condition.Name != "CostWasPaid" && condition.Name != "CostWasntPaid") {
		return nil, unsupported(condition, "payment condition")
	}
	onPaid := (continuation.Name == "If") == (condition.Name == "CostWasPaid")
	plan := &linkedExilePaymentPlan{payment.Position.String()}
	branch := ctx
	branch.linkedExilePayment = nil
	if onPaid {
		branch.linkedExilePayment = plan
	}
	parts, err := list(continuation.Args[1])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(continuation, "payment continuation requires at least one action")
	}
	body, err := compileActionSequence(parts, branch)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		result := mage.BindNumber("exiled at " + plan.position)
		bindings.linkedExilePayments = maps.Clone(bindings.linkedExilePayments)
		if bindings.linkedExilePayments == nil {
			bindings.linkedExilePayments = make(map[*linkedExilePaymentPlan]*mage.NumberBinding)
		}
		bindings.linkedExilePayments[plan] = result
		effects := make([]mage.Effect, len(body))
		for i, build := range body {
			effects[i] = build(bindings)
		}
		var paid = mage.Sequence(effects...)
		var unpaid mage.ResolvingEffect
		if !onPaid {
			paid, unpaid = unpaid, paid
		}
		return mage.PayChosenExileOrElse(mage.You, set(bindings), result, paid, unpaid)
	}, nil
}
