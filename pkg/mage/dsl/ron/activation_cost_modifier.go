package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileActivationCostModifier(n Node, ctx scope) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	selector, operation := n.Args[0], n.Args[1]
	if err := arity(selector, 1); err != nil {
		return nil, err
	}
	if err := arity(operation, 1); err != nil {
		return nil, err
	}
	switch operation.Name {
	case "IncreaseManaCost":
		if selector.Name != "AbilityOfAPermanent" {
			return nil, unsupported(selector, "activation cost increase subject")
		}
		filter, err := compileActivationCostFilter(selector.Args[0])
		if err != nil {
			return nil, err
		}
		parts, err := list(operation.Args[0])
		if err != nil {
			return nil, err
		}
		for _, part := range parts {
			if part.Name == "ManaCostX" {
				return nil, unsupported(part, "fixed activation cost increase")
			}
		}
		cost, err := manaCost(operation.Args[0])
		if err != nil {
			return nil, err
		}
		increase := core.ParseManaCost(cost)
		return func() mage.CardOption {
			return mage.WithStaticAbility(mage.IncreaseActivationCostStatic(filter, increase))
		}, nil
	case "ReduceManaCostNotLessThanOne":
		if selector.Name != "AbilityOfPermanent" || !ctx.aura || selector.Args[0].Kind != Symbol || selector.Args[0].Name != "HostPermanent" {
			return nil, unsupported(selector, "activation cost reduction subject")
		}
		parts, err := list(operation.Args[0])
		if err != nil {
			return nil, err
		}
		if len(parts) != 1 || parts[0].Name != "CostReduceGeneric" {
			return nil, problem(operation.Args[0], "supply one fixed generic reduction")
		}
		part := parts[0]
		if err := arity(part, 1); err != nil {
			return nil, err
		}
		if part.Args[0].Kind != IntegerNode || part.Args[0].Integer < 0 {
			return nil, problem(part.Args[0], "supply a nonnegative generic reduction")
		}
		amount := part.Args[0].Integer
		return func() mage.CardOption {
			return mage.WithStaticAbility(mage.ReduceActivationCostStatic(mage.AttachedPermanent, amount, 1))
		}, nil
	default:
		return nil, unsupported(operation, "activation cost modifier")
	}
}

func compileActivationCostFilter(n Node) (mage.PermanentFilter, error) {
	if err := arity(n, 1); err != nil {
		return mage.PermanentFilter{}, err
	}
	switch n.Name {
	case "IsColor", "IsCardtype":
		return compileFilter(n)
	case "And", "Or":
		parts, err := list(n.Args[0])
		if err != nil {
			return mage.PermanentFilter{}, err
		}
		if len(parts) == 0 {
			return mage.PermanentFilter{}, problem(n, "supply a nonempty activation source filter")
		}
		filters := make([]mage.PermanentFilter, len(parts))
		for i, part := range parts {
			filters[i], err = compileActivationCostFilter(part)
			if err != nil {
				return mage.PermanentFilter{}, err
			}
		}
		if n.Name == "And" {
			return mage.And(filters...), nil
		}
		return mage.Or(filters...), nil
	default:
		return mage.PermanentFilter{}, unsupported(n, "activation cost source filter")
	}
}
