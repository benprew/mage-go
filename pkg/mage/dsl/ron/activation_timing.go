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
		timing, err = compileActivationTiming(n.Args[2])
		if err != nil {
			return nil, err
		}
	}
	costs, hasX, err := compileCosts(n.Args[0])
	if err != nil {
		return nil, err
	}
	ctx.x = hasX
	plan, err := compileActions(n.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		options := actionOptions(plan.build())
		for _, build := range costs {
			options = append(options, mage.WithCost(build()))
		}
		options = append(options, timing...)
		return mage.WithAbility(mage.NewAction(mage.ActionActivated, options...))
	}, nil
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
