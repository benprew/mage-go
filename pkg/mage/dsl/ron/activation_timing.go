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
	if conditionalManaBody(n.Args[1]) {
		return compileConditionalMana(n)
	}
	var timing []mage.ActionOption
	if count == 3 {
		var err error
		timing, err = compileActivationModifiers(n.Args[2], ctx)
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
	exile, err := compileExilePaymentPlan(n.Args[0])
	if err != nil {
		return nil, err
	}
	ctx.exilePayment = exile
	costs, hasX, err := compileExileActivationCosts(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	ctx.x = hasX
	ctx.activation = true
	ctx.activationHistory = &activationHistoryPlan{}
	var plan actionPlan
	var modes []actionPlan
	if n.Args[1].Name == "Modal_ChooseOne" {
		modes, err = compileChooseOneModes(n.Args[1], ctx)
	} else {
		plan, err = compileActions(n.Args[1], ctx)
	}
	if err != nil {
		return nil, err
	}
	if ctx.activationHistory.used {
		if modes != nil {
			return nil, problem(n, "activation history in modes is not supported")
		}
		if containsManaAction(n.Args[1]) {
			if err := validateHistoryManaBody(n.Args[1]); err != nil {
				return nil, err
			}
		}
	}
	return func() mage.CardOption {
		bindings := instanceBindings{counters: make(map[*counterPaymentPlan]*mage.NumberBinding), sacrifices: make(map[*sacrificeValuePlan]sacrificeValueBindings)}
		if exile != nil {
			bindings.exiles = map[*exilePaymentPlan]*mage.NumberBinding{exile: mage.BindNumber("paid exile mana value")}
		}
		if paid != nil {
			bindings.sacrifices[paid] = sacrificeValueBindings{power: mage.BindNumber("paid sacrifice power"), value: mage.BindNumber("paid sacrifice value")}
		}
		if counters != nil {
			bindings.counters[counters] = mage.BindNumber("paid source counters")
		}
		if modes != nil {
			primary := mage.GenericCost(0)
			if len(costs) != 0 {
				primary = costs[0](bindings)
			}
			action := mage.NewModalActivated(primary, buildChooseOneModes(modes, bindings))
			for i, build := range costs {
				if i == 0 {
					continue
				}
				mage.WithCost(build(bindings))(action)
			}
			for _, option := range timing {
				option(action)
			}
			return mage.WithAbility(action)
		}
		if ctx.activationHistory.used {
			bindings.activationHistory = mage.BindActivationHistory("this activated ability")
		}
		options := actionOptions(plan.buildWithBindings(bindings))
		if ctx.activationHistory.used {
			options = append(options, mage.WithActivationHistory(bindings.activationHistory))
		}
		for _, build := range costs {
			options = append(options, mage.WithCost(build(bindings)))
		}
		options = append(options, timing...)
		return mage.WithAbility(mage.NewAction(mage.ActionActivated, options...))
	}, nil
}

func compileActivationModifiers(n Node, ctx scope) ([]mage.ActionOption, error) {
	seenMaximum := false
	seenSorcery := false
	seenPermission := false
	options, err := compileActivationModifier(n, ctx, &seenMaximum, &seenSorcery, &seenPermission)
	if err != nil {
		return nil, err
	}
	if seenSorcery {
		// Own-turn options must not replace the stronger timing restriction.
		options = append(options, mage.WithSorcerySpeed())
	}
	return options, nil
}

func compileActivationModifier(n Node, ctx scope, seenMaximum, seenSorcery, seenPermission *bool) ([]mage.ActionOption, error) {
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
			next, err := compileActivationModifier(item, ctx, seenMaximum, seenSorcery, seenPermission)
			if err != nil {
				return nil, err
			}
			options = append(options, next...)
		}
		return options, nil
	case "OnlyPlayerMayActivate":
		if *seenPermission {
			return nil, problem(n, "multiple activation permissions are not supported")
		}
		options, err := compileOnlyPlayerActivation(n, ctx)
		if err != nil {
			return nil, err
		}
		*seenPermission = true
		return options, nil
	case "OtherPlayersMayActivate":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		player := n.Args[0]
		if player.Kind != Symbol || player.Name != "AnyPlayer" {
			return nil, unsupported(player, "activation permission player filter")
		}
		if *seenPermission {
			return nil, problem(n, "multiple activation permissions are not supported")
		}
		*seenPermission = true
		return []mage.ActionOption{mage.WithAnyPlayerMay()}, nil
	case "ActivateOnlyOnceEachTurn":
		if _, err := symbol(n); err != nil {
			return nil, err
		}
		return []mage.ActionOption{mage.WithOncePerTurn()}, nil
	case "ActivateOnlyAsASorcery":
		if _, err := symbol(n); err != nil {
			return nil, err
		}
		*seenSorcery = true
		return nil, nil
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
		return compileActivationTiming(n, ctx)
	}
}

func compileActivationTiming(n Node, ctx scope) ([]mage.ActionOption, error) {
	if n.Name != "ActivateOnlyIf" {
		return nil, unsupported(n, "activation modifier")
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	filter := n.Args[0]
	if filter.Name == "PermanentPassesFilter" {
		if err := arity(filter, 2); err != nil {
			return nil, err
		}
		subject, err := compileConditionalSubject(filter.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		property, err := compileResolvingPropertyFilter(filter.Args[1])
		if err != nil {
			return nil, err
		}
		return []mage.ActionOption{mage.WithActivationStateCondition(mage.PermanentMatchesFilter(subject, property))}, nil
	}
	if filter.Kind == Symbol {
		if filter.Name == "IsBeforeCombatDamageStep" {
			return []mage.ActionOption{mage.WithActivationCondition(func(g *mage.Game, _ *mage.Permanent, _ uuid.UUID) bool {
				return g.CombatWindowOpen(core.CombatDamage, false)
			})}, nil
		}
		var steps []core.PhaseStep
		switch filter.Name {
		case "IsDuringCombat":
			steps = []core.PhaseStep{core.BeginCombat, core.DeclareAttackers, core.DeclareBlockers, core.FirstStrikeDamage, core.CombatDamage, core.EndCombat}
		case "IsDuringEndOfCombatStep":
			steps = []core.PhaseStep{core.EndCombat}
		case "IsDuringDeclareBlockersStep":
			steps = []core.PhaseStep{core.DeclareBlockers}
		case "IsDuringUpkeep":
			steps = []core.PhaseStep{core.Upkeep}
		default:
			return nil, unsupported(filter, "activation timing condition")
		}
		return []mage.ActionOption{mage.WithActivationCondition(func(g *mage.Game, _ *mage.Permanent, _ uuid.UUID) bool {
			for _, step := range steps {
				if g.GetStep() == step {
					return true
				}
			}
			return false
		})}, nil
	}
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
