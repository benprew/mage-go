package ron

import (
	"strconv"

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
	if containsDeclarationName(n.Args[0], "PayManaCostOfPermanent") && containsManaAction(n.Args[1]) {
		return nil, problem(n, "permanent mana cost requires a nonmana activation body")
	}
	if build, matched, err := compileDeferredMana(n, ctx); matched {
		return build, err
	}
	if containsCounterMana(n.Args[1]) {
		return compileSourceCounterMana(n)
	}
	if containsDeclarationName(n.Args[1], "AnyManaColorAPermanentCouldProduce") {
		return compileDynamicManaColors(n)
	}
	if conditionalManaBody(n.Args[1]) {
		return compileConditionalMana(n)
	}
	paid, err := compileSacrificeValuePlan(n.Args[0], n.Args[1])
	if err != nil {
		return nil, err
	}
	ctx.pluralSacrifice, err = compilePluralSacrificePlan(n.Args[0])
	if err != nil {
		return nil, err
	}
	ctx.sacrificeValue = paid
	ctx.sacrificeSubtype = activationSacrificeSubtypePlan(n.Args[0])
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
	ctx.sourceCardPayment = &sourceCardPaymentPlan{}
	ctx.discardFacts = &discardFactPlan{}
	if ctx.pluralSacrifice != nil && ctx.pluralSacrifice.source {
		ctx.sourceCardPayment.used = true
	}
	costs, hasX, err := compileExileActivationCosts(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	ctx.x = hasX
	var timing []mage.ActionOption
	if count == 3 {
		timing, err = compileActivationModifiers(n.Args[2], ctx)
		if err != nil {
			return nil, err
		}
		if containsDeclarationName(n.Args[2], "XIs") && containsDeclarationName(n.Args[1], "ValueX") {
			return nil, problem(n.Args[1], "defined mana X does not support ValueX in targets or resolving bodies")
		}
	}
	ctx.activation = true
	ctx.activationMana = &activationManaPlan{}
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
	if count == 3 && hasTargetDefinedX(n.Args[2]) && (plan.spellTarget == nil || len(plan.targets) != 1 || modes != nil) {
		return nil, problem(n.Args[2], "activation X requires its sole spell target")
	}
	if count == 3 && hasTargetDefinedX(n.Args[2]) {
		plan.activationSpellManaX = true
	}
	if result := ctx.sacrificeSubtype; result != nil && result.subtype != "" {
		if paid != nil {
			return nil, problem(n, "combined sacrifice characteristic and subtype results need a separate contract")
		}
		if err := arity(*result.cost, 1); err != nil {
			return nil, err
		}
		result.filter, err = compileSacrificeCostFilter(result.cost.Args[0])
		if err != nil {
			return nil, err
		}
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
	if ctx.activationMana.used {
		cost := n.Args[0]
		if cost.Name != "PayManaX" || len(cost.Args) != 2 || cost.Args[0].Kind != List || len(cost.Args[0].Args) != 1 || cost.Args[0].Args[0].Kind != Symbol || cost.Args[0].Args[0].Name != "ManaCostX" || cost.Args[1].Kind != Symbol || cost.Args[1].Name != "ValueX" {
			return nil, problem(cost, "mana subset requires one pure announced X payment")
		}
	}
	if ctx.activationMana.used && (modes != nil || containsManaAction(n.Args[1])) {
		return nil, problem(n, "mana receipt requires a nonmodal nonmana activation")
	}
	return func() mage.CardOption {
		bindings := instanceBindings{sacrificeSubtypes: make(map[*sacrificeSubtypePlan]*mage.NumberBinding), counters: make(map[*counterPaymentPlan]*mage.NumberBinding), sacrifices: make(map[*sacrificeValuePlan]sacrificeValueBindings)}
		bindings.discardFacts = map[*discardFactPlan][]*mage.NumberBinding{ctx.discardFacts: make([]*mage.NumberBinding, len(ctx.discardFacts.filters))}
		for i := range ctx.discardFacts.filters {
			bindings.discardFacts[ctx.discardFacts][i] = mage.BindNumber("discard payment fact " + strconv.Itoa(i))
		}
		if ctx.pluralSacrifice != nil {
			bindings.pluralSacrifices = map[*pluralSacrificePlan]pluralSacrificeBindings{ctx.pluralSacrifice: {power: mage.BindNumber("plural sacrifice power"), cards: mage.BindCards("plural sacrifice cards")}}
		}
		if ctx.sourceCardPayment.used {
			bindings.sourceCards = map[*sourceCardPaymentPlan]*mage.CardBinding{ctx.sourceCardPayment: mage.BindCard("paid source card")}
		}
		if exile != nil && !exile.singleGraveyard && exile.permanentFilter == nil {
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
			mage.WithLayeredTextRevisionActivationState()(action)
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
		if ctx.activationMana.used {
			bindings.activationMana = mage.BindActivationMana("activation mana paid")
		}
		if ctx.activationHistory.used {
			bindings.activationHistory = mage.BindActivationHistory("this activated ability")
		}
		actions := plan.buildWithBindings(bindings)
		options := actionOptions(actions)
		if ctx.activationMana.used {
			options = append(options, mage.WithActivationManaXReceipt(bindings.activationMana))
		}
		options = append(options, mage.WithLayeredTextRevisionActivationState())
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
	if countDeclarationName(n, "XCantBeZero") > 1 {
		return nil, problem(n, "multiple X minimum declarations are not supported")
	}
	if countDeclarationName(n, "XIs") > 1 {
		return nil, problem(n, "multiple X definitions are not supported")
	}
	options, err := compileActivationModifier(n, ctx, activationDuringCombat(n), &seenMaximum, &seenSorcery, &seenPermission)
	if err != nil {
		return nil, err
	}
	if seenSorcery {
		// Own-turn options must not replace the stronger timing restriction.
		options = append(options, mage.WithSorcerySpeed())
	}
	return options, nil
}

func compileActivationModifier(n Node, ctx scope, during bool, seenMaximum, seenSorcery, seenPermission *bool) ([]mage.ActionOption, error) {
	switch n.Name {
	case "XCantBeZero":
		if _, err := symbol(n); err != nil {
			return nil, err
		}
		if !ctx.x {
			return nil, problem(n, "XCantBeZero requires an X mana payment")
		}
		return []mage.ActionOption{mage.WithActivationMinimumX(1)}, nil
	case "XIs":
		return compileActivationManaX(n, ctx)
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
			next, err := compileActivationModifier(item, ctx, during, seenMaximum, seenSorcery, seenPermission)
			if err != nil {
				return nil, err
			}
			options = append(options, next...)
		}
		return options, nil
	case "OnlyOtherPlayersMayActivate":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		if n.Args[0].Kind != Symbol || n.Args[0].Name != "Opponent" {
			return nil, unsupported(n.Args[0], "activation permission player filter")
		}
		if *seenPermission {
			return nil, problem(n, "multiple activation permissions are not supported")
		}
		*seenPermission = true
		return []mage.ActionOption{mage.WithAnyPlayerMay(), mage.WithActivationStateCondition(mage.NotCondition(mage.PlayerIsPermanentController(mage.This)))}, nil
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
		return compileActivationTiming(n, ctx, during)
	}
}

func compileActivationTiming(n Node, ctx scope, during bool) ([]mage.ActionOption, error) {
	if n.Name != "ActivateOnlyIf" {
		return nil, unsupported(n, "activation modifier")
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	filter := n.Args[0]
	switch filter.Name {
	case "And", "IsAPlayersTurn", "IsPlayersTurn", "IsDuringCombat", "IsBeforeAttackersDeclared", "IsBeforeBlockersDeclared", "IsBeforeCombatDamageStep":
		condition, err := compileActivationWindow(filter, during)
		if err != nil {
			return nil, err
		}
		return []mage.ActionOption{mage.WithActivationStateCondition(condition)}, nil
	}
	if filter.Name == "PermanentPassesFilter" {
		if err := arity(filter, 2); err != nil {
			return nil, err
		}
		subject, err := compileConditionalSubject(filter.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		if filter.Args[1].Name == "ControlledSinceBeginningOfMostRecentTurn" {
			if _, err := symbol(filter.Args[1]); err != nil {
				return nil, err
			}
			return []mage.ActionOption{mage.WithActivationStateCondition(mage.ControlledSinceTurnStart(subject))}, nil
		}
		property, err := compileResolvingPropertyFilter(filter.Args[1])
		if err != nil {
			return nil, err
		}
		return []mage.ActionOption{mage.WithActivationStateCondition(mage.PermanentMatchesFilter(subject, property))}, nil
	}
	if filter.Kind == Symbol {
		var steps []core.PhaseStep
		switch filter.Name {
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
	if filter.Args[1].Name == "NumCardsInHandIs" {
		count, err := compileActivationHandCount(filter.Args[1])
		if err != nil {
			return nil, err
		}
		return []mage.ActionOption{mage.WithActivationStateCondition(mage.ControllerHandSizeIs(count))}, nil
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

func compileActivationHandCount(n Node) (int, error) {
	if err := arity(n, 1); err != nil {
		return 0, err
	}
	comparison := n.Args[0]
	if comparison.Name != "EqualTo" {
		return 0, unsupported(comparison, "activation hand comparison")
	}
	if err := arity(comparison, 1); err != nil {
		return 0, err
	}
	value := comparison.Args[0]
	if value.Name != "Integer" {
		return 0, unsupported(value, "activation hand count")
	}
	if err := arity(value, 1); err != nil {
		return 0, err
	}
	if value.Args[0].Kind != IntegerNode || value.Args[0].Integer < 0 {
		return 0, problem(value, "hand count requires a nonnegative integer literal")
	}
	return value.Args[0].Integer, nil
}
