package ron

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileActivatedRule(n Node, ctx scope) (optionFactory, error) {
	count := 2
	if n.Name == "ActivatedWithModifiers" {
		count = 3
	}
	if err := arity(n, count); err != nil {
		return nil, err
	}
	var timing []mage.ActionOption
	if count == 3 {
		var err error
		timing, err = compileActivationModifiers(n.Args[2])
		if err != nil {
			return nil, err
		}
	}
	paid, err := compileSacrificeValuePlan(n.Args[0], n.Args[1])
	if err != nil {
		return nil, err
	}
	ctx.sacrificeValue = paid
	counters, err := compileCounterPaymentPlan(n.Args[0])
	if err != nil {
		return nil, err
	}
	ctx.counterPayment = counters
	costs, hasX, err := compileActivationCosts(n.Args[0], paid, counters)
	if err != nil {
		return nil, err
	}
	ctx.x = hasX
	plan, err := compileActions(n.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		bindings := instanceBindings{counters: make(map[*counterPaymentPlan]*mage.NumberBinding), sacrifices: make(map[*sacrificeValuePlan]sacrificeValueBindings)}
		if paid != nil {
			bindings.sacrifices[paid] = sacrificeValueBindings{power: mage.BindNumber("paid sacrifice power"), value: mage.BindNumber("paid sacrifice value")}
		}
		if counters != nil {
			bindings.counters[counters] = mage.BindNumber("paid source counters")
		}
		options := actionOptions(plan.buildWithBindings(bindings))
		for _, build := range costs {
			options = append(options, mage.WithCost(build(bindings)))
		}
		options = append(options, timing...)
		return mage.WithAbility(mage.NewAction(mage.ActionActivated, options...))
	}, nil
}

func compileActivationModifiers(n Node) ([]mage.ActionOption, error) {
	seenMaximum := false
	return compileActivationModifier(n, &seenMaximum)
}

func compileActivationModifier(n Node, seenMaximum *bool) ([]mage.ActionOption, error) {
	switch n.Name {
	case "And":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		items, err := list(n.Args[0])
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			return nil, unsupported(n, "empty activation modifiers")
		}
		var options []mage.ActionOption
		for _, item := range items {
			next, err := compileActivationModifier(item, seenMaximum)
			if err != nil {
				return nil, err
			}
			options = append(options, next...)
		}
		return options, nil
	case "ActivateOnlyOnceEachTurn":
		if _, err := symbol(n); err != nil {
			return nil, err
		}
		return []mage.ActionOption{mage.WithOncePerTurn()}, nil
	case "ActivateNoMoreThanNumberTimesEachTurn":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		value := n.Args[0]
		if value.Name != "Integer" {
			return nil, unsupported(value, "fixed activation limit")
		}
		if err := arity(value, 1); err != nil {
			return nil, err
		}
		if value.Args[0].Kind != IntegerNode {
			return nil, problem(value, "expected integer literal")
		}
		count := value.Args[0].Integer
		if count <= 0 {
			return nil, unsupported(value, "positive activation limit")
		}
		if *seenMaximum {
			return nil, problem(n, "multiple maximum activation limits are not supported")
		}
		*seenMaximum = true
		return []mage.ActionOption{mage.WithMaxActivationsPerTurn(count)}, nil
	default:
		return compileActivationTiming(n)
	}
}

func compileActivationTiming(n Node) ([]mage.ActionOption, error) {
	if n.Name != "ActivateOnlyIf" {
		return nil, unsupported(n, "activation modifier")
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	filter := n.Args[0]
	if filter.Name != "PlayerPassesFilter" {
		return nil, unsupported(filter, "activation timing condition")
	}
	if err := arity(filter, 2); err != nil {
		return nil, err
	}
	player, err := symbol(filter.Args[0])
	if err != nil {
		return nil, err
	}
	if player != "You" {
		return nil, unsupported(filter.Args[0], "activation timing player")
	}
	condition, err := symbol(filter.Args[1])
	if err != nil {
		return nil, err
	}
	switch condition {
	case "IsTheirTurn":
		return []mage.ActionOption{mage.WithYourTurnOnly()}, nil
	case "IsTheirUpkeep":
		return []mage.ActionOption{mage.WithYourTurnOnly(), mage.WithActivationCondition(func(g *mage.Game, _ *mage.Permanent, _ uuid.UUID) bool { return g.GetStep() == core.Upkeep })}, nil
	default:
		return nil, unsupported(filter.Args[1], "activation timing filter")
	}
}
