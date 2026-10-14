package ron

import "github.com/benprew/mage-go/pkg/mage"

type permanentExileFactPlan struct {
	filter mage.PermanentFilter
	label  string
}

type permanentExilePaymentPlan struct {
	filter mage.PermanentFilter
	facts  []*permanentExileFactPlan
}

func compilePermanentExilePayment(cost Node) (*permanentExilePaymentPlan, error) {
	if cost.Name != "Exile" {
		return nil, nil
	}
	plan, err := compileExileCost(cost)
	if err != nil || plan == nil || plan.permanentFilter == nil {
		return nil, err
	}
	return &permanentExilePaymentPlan{filter: *plan.permanentFilter}, nil
}

func compilePermanentExileFactFilter(n Node) (mage.PermanentFilter, error) {
	if n.Name == "And" || n.Name == "Or" {
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return mage.PermanentFilter{}, err
		}
		if len(parts) == 0 {
			return mage.PermanentFilter{}, problem(n, "payment fact requires a nonempty filter")
		}
		filters := make([]mage.PermanentFilter, len(parts))
		for i, part := range parts {
			filters[i], err = compilePermanentExileFactFilter(part)
			if err != nil {
				return mage.PermanentFilter{}, err
			}
		}
		if n.Name == "And" {
			return mage.And(filters...), nil
		}
		return mage.Or(filters...), nil
	}
	if n.Kind == Symbol && n.Name == "IsNonToken" {
		return compileFilter(n)
	}
	switch n.Name {
	case "IsCardtype", "IsNonCardtype", "IsColor", "IsNonColor":
		return compileFilter(n)
	case "IsCreatureType", "IsNonCreatureType", "IsLandType", "IsNonLandType", "IsEnchantmentType":
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, err
		}
		subtype, err := symbol(n.Args[0])
		if err != nil {
			return mage.PermanentFilter{}, err
		}
		family := mage.SubtypeCreature
		switch n.Name {
		case "IsLandType", "IsNonLandType":
			family = mage.SubtypeLand
		case "IsEnchantmentType":
			family = mage.SubtypeEnchantment
		}
		if err := mage.ValidateSubtypeFamily(family, subtypeName(subtype)); err != nil {
			return mage.PermanentFilter{}, problem(n, err.Error())
		}
		return compileFilter(n)
	default:
		return mage.PermanentFilter{}, unsupported(n, "permanent exile payment fact filter")
	}
}

func compilePermanentExileFactBranch(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	condition := n.Args[0]
	if err := arity(condition, 1); err != nil {
		return nil, err
	}
	if ctx.exileFacts == nil {
		return nil, problem(condition, "permanent exile fact requires a local additional casting payment")
	}
	filter, err := compilePermanentExileFactFilter(condition.Args[0])
	if err != nil {
		return nil, err
	}
	fact := &permanentExileFactPlan{filter: filter, label: condition.Position.String()}
	ctx.exileFacts.facts = append(ctx.exileFacts.facts, fact)
	parts, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(n, "permanent exile fact branch requires actions")
	}
	body, err := compileActionSequence(parts, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		effects := make([]mage.Effect, len(body))
		for i, build := range body {
			effects[i] = build(bindings)
		}
		return mage.IfPositive(mage.FromNumber(bindings.permanentExileFacts[fact]), mage.Sequence(effects...))
	}, nil
}

func buildPermanentExileFacts(plan *permanentExilePaymentPlan) map[*permanentExileFactPlan]*mage.NumberBinding {
	if plan == nil {
		return nil
	}
	bindings := make(map[*permanentExileFactPlan]*mage.NumberBinding, len(plan.facts))
	for _, fact := range plan.facts {
		bindings[fact] = mage.BindNumber("permanent exile payment fact " + fact.label)
	}
	return bindings
}

func buildPermanentExileFactCost(plan *permanentExilePaymentPlan, bindings instanceBindings) mage.Cost {
	facts := make([]mage.PermanentPaymentFact, len(plan.facts))
	for i, fact := range plan.facts {
		facts[i] = mage.PermanentPaymentFact{Filter: fact.filter, Result: bindings.permanentExileFacts[fact]}
	}
	return mage.ExileMatchingPermanentCostWithFacts(plan.filter, "Exile a matching permanent you control", facts...)
}
