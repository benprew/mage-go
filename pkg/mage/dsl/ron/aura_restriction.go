package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileAuraRestriction(n, operation Node, ctx scope) (optionFactory, error) {
	if err := arity(operation, 1); err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if subject.kind != permanentSource && subject.kind != permanentHost {
		return nil, unsupported(n.Args[0], "static Aura restriction reference")
	}
	filter := operation.Args[0]
	if filter.Name != "And" {
		return nil, unsupported(filter, "other Aura filter")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	parts, err := list(filter.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) != 2 {
		return nil, unsupported(filter, "other Aura filter")
	}
	other, aura := false, false
	for _, part := range parts {
		if err := arity(part, 1); err != nil {
			return nil, err
		}
		switch part.Name {
		case "Other":
			if other || part.Args[0].Kind != Symbol || part.Args[0].Name != "ThisPermanent" {
				return nil, unsupported(part, "Aura exception")
			}
			other = true
		case "IsEnchantmentType":
			if aura || part.Args[0].Kind != Symbol || part.Args[0].Name != "Aura" {
				return nil, unsupported(part, "Aura restriction subtype")
			}
			aura = true
		default:
			return nil, unsupported(part, "other Aura filter")
		}
	}
	if !other || !aura {
		return nil, unsupported(filter, "other Aura filter")
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.Static(mage.CantBeEnchantedExceptBy(subject.build(instanceBindings{}), mage.This)))
	}, nil
}

func compileCombinedAuraRestrictions(n Node, operations []Node, ctx scope) (optionFactory, error) {
	if len(operations) != 2 {
		return nil, unsupported(n.Args[1], "combined Aura restrictions")
	}
	options := make([]optionFactory, 0, len(operations))
	seen := make(map[string]bool)
	for _, operation := range operations {
		if seen[operation.Name] {
			return nil, problem(operation, "duplicate Aura restriction")
		}
		seen[operation.Name] = true
		var option optionFactory
		var err error
		switch operation.Name {
		case "CantBeTheTargetOfSpells":
			option, err = compileSpellTargetRestriction(n, operation, ctx)
		case "CantBeEnchantedByAnEnchantment":
			option, err = compileAuraRestriction(n, operation, ctx)
		default:
			return nil, unsupported(operation, "combined Aura restriction")
		}
		if err != nil {
			return nil, err
		}
		options = append(options, option)
	}
	return func() mage.CardOption {
		built := make([]mage.CardOption, len(options))
		for i, build := range options {
			built[i] = build()
		}
		return func(card *mage.BaseCard) {
			for _, option := range built {
				option(card)
			}
		}
	}, nil
}
