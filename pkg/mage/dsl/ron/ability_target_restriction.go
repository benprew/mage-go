package ron

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileAbilityTargetRestriction(n, operation Node, ctx scope) (optionFactory, error) {
	if err := arity(operation, 1); err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if subject.kind != permanentSource && subject.kind != permanentHost {
		return nil, unsupported(n.Args[0], "static ability targeting reference")
	}
	filter, err := compileAbilityTargetingFilter(operation.Args[0])
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.Static(mage.CantBeTargetedByAbilities(subject.build(instanceBindings{}), filter)))
	}, nil
}

func compileAbilityTargetingFilter(n Node) (mage.CardFilter, error) {
	if n.Name == "CanTargetOnly" {
		return compileTargetOnlySubtypeFilter(n)
	}
	if n.Name != "AbilityOfASource" {
		return mage.CardFilter{}, unsupported(n, "static ability filter")
	}
	if err := arity(n, 1); err != nil {
		return mage.CardFilter{}, err
	}
	return compileAbilitySourceFilter(n.Args[0])
}

func compileAbilitySourceFilter(n Node) (mage.CardFilter, error) {
	if err := arity(n, 1); err != nil {
		return mage.CardFilter{}, err
	}
	switch n.Name {
	case "IsCardtype":
		kind, err := cardType(n.Args[0])
		if err != nil {
			return mage.CardFilter{}, err
		}
		return mage.NewCardFilter(kind.String()+" source", func(card mage.Card) bool { return card.HasType(kind) }), nil
	case "IsColor":
		value, err := color(n.Args[0])
		if err != nil {
			return mage.CardFilter{}, err
		}
		if value == core.Colorless {
			return mage.CardFilter{}, problem(n.Args[0], "ability source color requires one of the five colors")
		}
		return mage.HasColorCardFilter(value), nil
	case "And", "Or":
		parts, err := list(n.Args[0])
		if err != nil {
			return mage.CardFilter{}, err
		}
		if len(parts) == 0 {
			return mage.CardFilter{}, problem(n, "ability source filter requires at least one operand")
		}
		filters := make([]mage.CardFilter, len(parts))
		for i, part := range parts {
			filters[i], err = compileAbilitySourceFilter(part)
			if err != nil {
				return mage.CardFilter{}, err
			}
		}
		if n.Name == "And" {
			return mage.AndCardFilter(filters...), nil
		}
		return mage.NewCardFilter("matching ability source", func(card mage.Card) bool {
			return slices.ContainsFunc(filters, func(filter mage.CardFilter) bool { return filter.Match(card) })
		}), nil
	default:
		return mage.CardFilter{}, unsupported(n, "ability source filter")
	}
}
