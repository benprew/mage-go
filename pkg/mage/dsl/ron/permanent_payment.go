package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

type permanentPaymentPlan struct{ name string }

func compilePermanentPayment(n Node, remaining []Node, ctx scope) (effectFactory, error) {
	if ctx.drawReplacementBody || ctx.enteringSource {
		return nil, problem(n, "permanent payment requires an ordinary resolving body")
	}
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	subjects, err := compileContextualCollection(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if n.Args[1].Kind != Symbol || n.Args[1].Name != "AnyPlayer" {
		return nil, unsupported(n.Args[1], "permanent payment players")
	}
	cost := n.Args[2]
	if cost.Name != "PayLife" {
		return nil, unsupported(cost, "permanent payment cost")
	}
	if err := arity(cost, 1); err != nil {
		return nil, err
	}
	amount := cost.Args[0]
	if err := arity(amount, 1); err != nil {
		return nil, err
	}
	if amount.Name != "Integer" || amount.Args[0].Kind != IntegerNode || amount.Args[0].Integer < 0 {
		return nil, problem(amount, "permanent payment requires nonnegative fixed life")
	}
	plan := &permanentPaymentPlan{name: "unpaid permanents " + n.Position.String()}
	ctx.permanentPayment = plan
	body, err := compileActionSequence(remaining, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		bindings.permanentPayments = maps.Clone(bindings.permanentPayments)
		if bindings.permanentPayments == nil {
			bindings.permanentPayments = make(map[*permanentPaymentPlan]*mage.PermanentSetBinding)
		}
		result := mage.BindPermanentSet(plan.name)
		bindings.permanentPayments[plan] = result
		effects := make([]mage.Effect, 0, 1+len(body))
		effects = append(effects, mage.OfferCostForEachPermanent(subjects(bindings), mage.EachPlayer, mage.Costs(mage.LifePayCost(amount.Args[0].Integer)), result))
		for _, build := range body {
			effects = append(effects, build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}
