package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileResolvingPermanentBranch(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if ctx.activation && containsManaAction(n) {
		return nil, problem(n, "conditional mana activation requires classification and solver planning")
	}
	condition := n.Args[0]
	if err := arity(condition, 2); err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(condition.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	filterNode := condition.Args[1]
	combat := filterNode.Kind == Symbol && (filterNode.Name == "IsAttacking" || filterNode.Name == "IsBlocking")
	var filter mage.PermanentFilter
	if !combat {
		filter, err = compileResolvingPropertyFilter(filterNode)
		if err != nil {
			return nil, err
		}
	}
	parts, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(n.Args[1], "condition requires actions")
	}
	effects, err := compileActionSequence(parts, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		body := make([]mage.Effect, len(effects))
		for i, build := range effects {
			body[i] = build(bindings)
		}
		sequence := mage.Sequence(body...)
		ref := subject.build(bindings)
		if combat {
			if filterNode.Name == "IsAttacking" {
				return mage.IfPermanentAttacking(ref, sequence)
			}
			return mage.IfPermanentBlocking(ref, sequence)
		}
		return mage.IfPermanentMatches(ref, filter, sequence)
	}, nil
}

func compileResolvingPropertyFilter(n Node) (mage.PermanentFilter, error) {
	switch n.Name {
	case "HasACounterOfType", "HasNoCountersOfType", "HasNumberCountersOfType":
		return compileResolvingCounterFilter(n)
	case "And", "Or":
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return mage.PermanentFilter{}, err
		}
		if len(parts) == 0 {
			return mage.PermanentFilter{}, problem(n, "property filter requires operands")
		}
		filters := make([]mage.PermanentFilter, len(parts))
		for i, part := range parts {
			filters[i], err = compileResolvingPropertyFilter(part)
			if err != nil {
				return mage.PermanentFilter{}, err
			}
		}
		if n.Name == "And" {
			return mage.And(filters...), nil
		}
		return mage.Or(filters...), nil
	case "AnyPermanent", "IsTapped", "IsUntapped", "IsSupertype", "IsNonSupertype", "IsCardtype", "IsNonCardtype", "IsCreatureType", "IsNonCreatureType", "IsLandType", "IsNonLandType", "IsEnchantmentType", "IsColor", "IsNonColor", "HasAbility", "DoesntHaveAbility", "PowerIs", "ToughnessIs", "PowerAndToughnessIsEqualTo":
		return compileFilter(n)
	default:
		return mage.PermanentFilter{}, unsupported(n, "resolving permanent property filter")
	}
}
