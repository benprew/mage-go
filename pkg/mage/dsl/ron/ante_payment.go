package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileAntePayment(payment, continuation Node, ctx scope) (effectFactory, error) {
	count := 1
	if payment.Name == "PlayerMayCost" {
		count = 2
	}
	if err := arity(payment, count); err != nil {
		return nil, err
	}
	payer := ctx.player
	if count == 2 {
		var err error
		payer, err = compilePlayerRef(payment.Args[0], ctx)
		if err != nil {
			return nil, err
		}
	}
	cost := payment.Args[count-1]
	top := cost.Name == "AnteTopCardOfLibrary"
	var subject permanentReferencePlan
	if top {
		if cost.Kind != Symbol {
			return nil, problem(cost, "AnteTopCardOfLibrary requires a symbol")
		}
		if payment.Name == "MustCost" {
			return nil, unsupported(payment, "mandatory library-top ante payment")
		}
	} else {
		if err := arity(cost, 1); err != nil {
			return nil, err
		}
		ref := cost.Args[0]
		if ref.Name != "SinglePermanent" {
			return nil, unsupported(ref, "ante payment subject")
		}
		if err := arity(ref, 1); err != nil {
			return nil, err
		}
		var err error
		subject, err = compilePermanentRef(ref.Args[0], ctx)
		if err != nil {
			return nil, err
		}
	}
	branchCount := 2
	if continuation.Name == "IfElse" {
		branchCount = 3
	} else if continuation.Name != "If" && continuation.Name != "Unless" {
		return nil, problem(continuation, "ante payment requires an adjacent result continuation")
	}
	if err := arity(continuation, branchCount); err != nil {
		return nil, err
	}
	condition := continuation.Args[0]
	if condition.Kind != Symbol || condition.Name != "CostWasPaid" && condition.Name != "CostWasntPaid" {
		return nil, unsupported(condition, "ante payment condition")
	}
	branches := make([][]effectFactory, branchCount-1)
	for i := range branches {
		parts, err := list(continuation.Args[i+1])
		if err != nil {
			return nil, err
		}
		if len(parts) == 0 {
			return nil, problem(continuation.Args[i+1], "ante payment branch requires at least one action")
		}
		branches[i], err = compileActionSequence(parts, ctx)
		if err != nil {
			return nil, err
		}
	}
	return func(bindings instanceBindings) mage.Effect {
		built := make([]mage.ResolvingEffect, 2)
		for i, branch := range branches {
			effects := make([]mage.Effect, len(branch))
			for j, build := range branch {
				effects[j] = build(bindings)
			}
			built[i] = mage.Sequence(effects...)
		}
		if (continuation.Name == "Unless") == (condition.Name == "CostWasPaid") {
			built[0], built[1] = built[1], built[0]
		}
		if top {
			if len(branches) == 1 {
				if built[0] != nil {
					return mage.IfPaid(payer.build(bindings), mage.Costs(mage.AnteTopCost()), built[0])
				}
				return mage.UnlessPaid(payer.build(bindings), mage.Costs(mage.AnteTopCost()), built[1])
			}
			return mage.PayOrElse(payer.build(bindings), mage.Costs(mage.AnteTopCost()), built[0], built[1])
		}
		if payment.Name == "MustCost" {
			return mage.AntePermanentOrElse(payer.build(bindings), subject.build(bindings), built[0], built[1])
		}
		return mage.PayPermanentAnteOrElse(payer.build(bindings), subject.build(bindings), built[0], built[1])
	}, nil
}
