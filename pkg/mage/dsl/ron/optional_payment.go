package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileActionSequence(parts []Node, ctx scope) ([]effectFactory, error) {
	effects, _, err := compileActionSequenceWithName(parts, ctx)
	return effects, err
}

func compileActionSequenceWithName(parts []Node, ctx scope) ([]effectFactory, *cardNameChoicePlan, error) {
	var effects []effectFactory
	for i := 0; i < len(parts); i++ {
		part := parts[i]
		if part.Name == "ForEachPermanentAnyPlayerMayCost" {
			build, err := compilePermanentPayment(part, parts[i+1:], ctx)
			if err != nil {
				return nil, nil, err
			}
			return append(effects, build), ctx.chosenName, nil
		}
		if part.Name == "PlayerMustCost" {
			if i+1 == len(parts) {
				return nil, nil, problem(part, "payment requires an adjacent payment continuation")
			}
			effect, err := compileRequiredPlayerPayment(part, parts[i+1], ctx)
			if err != nil {
				return nil, nil, err
			}
			effects = append(effects, effect)
			i++
			continue
		}
		if part.Name == "EachPlayerAction" && part.Kind == Tuple && len(part.Args) == 2 && part.Args[1].Name == "ChooseUptoNumberPermanentsForEach" {
			build, err := compilePermanentSetChoice(part, parts[i+1:], ctx)
			if err != nil {
				return nil, nil, err
			}
			return append(effects, build), ctx.chosenName, nil
		}
		if part.Name == "EachPlayerAction" && part.Kind == Tuple && len(part.Args) == 2 && part.Args[1].Name == "TapAnyNumberOfPermanents" {
			build, err := compilePlayerTapResults(part, parts[i+1:], ctx)
			if err != nil {
				return nil, nil, err
			}
			return append(effects, build), ctx.chosenName, nil
		}
		if part.Name == "EachPlayerAction" && part.Kind == Tuple && len(part.Args) == 2 && part.Args[1].Name == "DiscardUptoNumberCards" {
			build, err := compilePlayerDiscardResults(part, parts[i+1:], ctx)
			if err != nil {
				return nil, nil, err
			}
			return append(effects, build), ctx.chosenName, nil
		}
		if part.Name == "SacrificeAPermanent" || part.Name == "SacrificeAPermanentOfAPlayersChoice" {
			result := &sacrificeSubtypePlan{label: part.Position.String()}
			build, err := compileChosenSacrificeWithResult(part, ctx, result)
			if err != nil {
				return nil, nil, err
			}
			ctx.sacrificeSubtype = result
			effects = append(effects, build)
			continue
		}
		if part.Name == "Exile" && part.Kind == Tuple && len(part.Args) == 2 && part.Args[0].Kind == List && len(part.Args[0].Args) > 1 {
			effect, err := compileGroupedExile(part, parts[i+1:], ctx)
			if err != nil {
				return nil, nil, err
			}
			return append(effects, effect), ctx.chosenName, nil
		}
		action, player, err := drawnCardAction(part, ctx)
		if err != nil {
			return nil, nil, err
		}
		if action.Name == "DrawACard" || action.Name == "RevealTheCardDrawnThisWay" {
			effect, err := compileDrawnCardProducer(action, player, parts[i+1:], ctx)
			if err != nil {
				return nil, nil, err
			}
			return append(effects, effect), ctx.chosenName, nil
		}
		if isRandomHandReveal(part) {
			effect, err := compileRandomHandReveal(part, parts[i+1:], ctx)
			if err != nil {
				return nil, nil, err
			}
			return append(effects, effect), ctx.chosenName, nil
		}

		if part.Name == "ChooseTwoColorWords" || part.Name == "ChooseTwoBasicLandTypes" {
			effect, result, err := compileTextWordChoice(part, ctx)
			if err != nil {
				return nil, nil, err
			}
			if result.color {
				ctx.colorWords = result
			} else {
				ctx.landWords = result
			}
			effects = append(effects, effect)
			continue
		}
		if part.Name == "ChooseACardName" {
			effect, result, err := compileCardNameChoiceWithResult(part, ctx)
			if err != nil {
				return nil, nil, err
			}
			ctx.chosenName = result
			effects = append(effects, effect)
			continue
		}
		if action.Name == "RevealTopCardOfLibrary" {
			effect, result, err := compileLibraryTopReveal(action, player)
			if err != nil {
				return nil, nil, err
			}
			ctx.libraryReveal = result
			ctx.handReveal = nil
			effects = append(effects, effect)
			continue
		}
		if part.Name == "PlayerActions" {
			effect, result, reveal, err := compilePlayerActionsWithResults(part, ctx)
			if err != nil {
				return nil, nil, err
			}
			ctx.chosenName = result
			if reveal != ctx.libraryReveal {
				ctx.handReveal = nil
			}
			ctx.libraryReveal = reveal
			effects = append(effects, effect)
			continue
		}
		if part.Name == "ChooseAPlayer" {
			effect, err := compilePlayerChoice(part, parts[i+1:], ctx)
			if err != nil {
				return nil, nil, err
			}
			return append(effects, effect), ctx.chosenName, nil
		}
		if part.Name == "PlayerAction" && part.Kind == Tuple && len(part.Args) == 2 && part.Args[1].Name == "ChooseAPermanent" {
			player, err := compilePlayerRef(part.Args[0], ctx)
			if err != nil {
				return nil, nil, err
			}
			choice := part.Args[1]
			if err := arity(choice, 1); err != nil {
				return nil, nil, err
			}
			choiceCtx := ctx
			choiceCtx.player = player
			subjects, err := compileContextualCollection(choice.Args[0], choiceCtx)
			if err != nil {
				return nil, nil, err
			}
			plan := &permanentChoicePlan{name: "chosen permanent " + choice.Position.String()}
			effect, err := compilePermanentChoiceContinuation(parts[i+1:], ctx, plan, player, subjects)
			if err != nil {
				return nil, nil, err
			}
			return append(effects, effect), ctx.chosenName, nil
		}
		if part.Name == "SimultaneousPermanentChoices" {
			effect, err := compileAPNAPPermanentChoices(part, parts[i+1:], ctx)
			if err != nil {
				return nil, nil, err
			}
			return append(effects, effect), ctx.chosenName, nil
		}
		if part.Name == "ChooseAPermanent" {
			effect, err := compilePermanentChoice(part, parts[i+1:], ctx)
			if err != nil {
				return nil, nil, err
			}
			return append(effects, effect), ctx.chosenName, nil
		}
		if part.Name == "ChooseADamageSource" {
			effect, err := compileGeneralDamageSource(part, parts[i+1:], ctx)
			if err != nil {
				return nil, nil, err
			}
			return append(effects, effect), ctx.chosenName, nil
		}
		if part.Name == "CreateValueX" {
			effect, err := compileCreatedValue(part, parts[i+1:], ctx)
			if err != nil {
				return nil, nil, err
			}
			return append(effects, effect), ctx.chosenName, nil
		}
		if part.Name == "ChooseABasicLandType" {
			if i+1 == len(parts) {
				return nil, nil, problem(part, "basic land choice requires an adjacent landwalk continuation")
			}
			effect, err := compileBasicLandChoice(part, parts[i+1], ctx)
			if err != nil {
				return nil, nil, err
			}
			effects = append(effects, effect)
			i++
			continue
		}
		if part.Name == "ChooseAColor" || part.Name == "ChooseColors" {
			if i+1 == len(parts) {
				return nil, nil, problem(part, "color choice requires an adjacent color-setting continuation")
			}
			effect, err := compileColorChoice(part, parts[i+1], ctx)
			if err != nil {
				return nil, nil, err
			}
			effects = append(effects, effect)
			i++
			continue
		}
		if (part.Name == "DestroyEachPermanent" || part.Name == "DestroyEachPermanentNoRegen") && i+1 < len(parts) && hasDestructionCount(parts[i+1]) {
			effect, err := compileDestructionCount(part, parts[i+1], ctx)
			if err != nil {
				return nil, nil, err
			}
			effects = append(effects, effect)
			i++
			continue
		}
		if part.Name == "ChooseAnAbility" {
			if i+1 == len(parts) {
				return nil, nil, problem(part, "ability choice requires an adjacent layer continuation")
			}
			effect, err := compileAbilityChoice(part, parts[i+1], ctx)
			if err != nil {
				return nil, nil, err
			}
			effects = append(effects, effect)
			i++
			continue
		}
		if isPaidUntapChoice(part) {
			if i+1 == len(parts) {
				return nil, nil, problem(part, "paid choice requires an adjacent untap of its result")
			}
			effect, err := compilePaidUntap(part, parts[i+1], ctx)
			if err != nil {
				return nil, nil, err
			}
			effects = append(effects, effect)
			i++
			continue
		}
		if part.Name == "EachPlayerAction" && part.Kind == Tuple && len(part.Args) == 2 && part.Args[1].Name == "ShuffleHandIntoLibrary" {
			if i+1 == len(parts) {
				return nil, nil, problem(part, "hand shuffle requires an adjacent all-player draw of its result")
			}
			effect, err := compileHandShuffleDraw(part, parts[i+1])
			if err != nil {
				return nil, nil, err
			}
			effects = append(effects, effect)
			i++
			continue
		}
		if part.Name == "PlayerMayCost" || part.Name == "MayCost" || part.Name == "MustCost" || part.Name == "AnyPlayerMayCost" {
			if i+1 == len(parts) {
				return nil, nil, problem(part, "payment requires an adjacent payment continuation")
			}
			effect, err := compileResolutionPayment(part, parts[i+1], ctx)
			if err != nil {
				return nil, nil, err
			}
			effects = append(effects, effect)
			i++
			continue
		}
		compiled, err := compileEffect(part, ctx)
		if err != nil {
			return nil, nil, err
		}
		effects = append(effects, compiled...)
	}
	return effects, ctx.chosenName, nil
}

func compileResolutionPayment(payment, continuation Node, ctx scope) (effectFactory, error) {
	if payment.Name == "AnyPlayerMayCost" {
		return compileEligiblePlayerPayment(payment, continuation, ctx)
	}
	if len(payment.Args) > 0 {
		cost := payment.Args[len(payment.Args)-1]
		if cost.Name == "AnteAPermanent" || cost.Name == "AnteTopCardOfLibrary" {
			return compileAntePayment(payment, continuation, ctx)
		}
	}
	if isResolutionActionPayment(payment) {
		return compileResolutionActionPayment(payment, continuation, ctx)
	}
	if len(payment.Args) > 0 && payment.Args[len(payment.Args)-1].Name == "RemoveCounters" {
		return compileResolutionCounterPayment(payment, continuation, ctx)
	}
	if payment.Name == "MustCost" && len(payment.Args) == 1 && payment.Args[0].Name == "TapPermanent" {
		return compileRequiredTapPayment(payment, continuation, ctx)
	}
	if continuation.Name == "IfElse" {
		return compilePaymentBranches(payment, continuation, ctx)
	}
	if payment.Name == "MustCost" && len(payment.Args) == 1 && payment.Args[0].Name == "DestroyPermanent" {
		return compileDestructionPayment(payment, continuation, ctx)
	}
	var player playerReferencePlan
	payer := Node{Kind: Symbol, Name: "You"}
	var cost Node
	if payment.Name == "MayCost" || payment.Name == "MustCost" {
		if err := arity(payment, 1); err != nil {
			return nil, err
		}
		cost = payment.Args[0]
	} else {
		if err := arity(payment, 2); err != nil {
			return nil, err
		}
		var err error
		player, err = compilePlayerRef(payment.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		cost = payment.Args[1]
		payer = payment.Args[0]
	}
	var sacrifice *permanentReferencePlan
	var dynamic valueFactory
	var life valueFactory
	var mana core.ManaCost
	var costs []costFactory
	var err error
	if cost.Name == "SacrificePermanent" {
		if err := arity(cost, 1); err != nil {
			return nil, err
		}
		ref, compileErr := compilePermanentRef(cost.Args[0], ctx)
		err = compileErr
		sacrifice = &ref
	} else if payment.Name == "MustCost" {
		var build costFactory
		build, err = compileRequiredSacrificeCost(cost, ctx)
		costs = []costFactory{build}
	} else if cost.Name == "PutCounters" {
		var build costFactory
		build, err = compileCounterPlacementPayment(cost, payer, ctx)
		costs = []costFactory{build}
	} else if cost.Name == "MillNumberCards" {
		var amount int
		amount, err = compileMillCount(cost, true)
		costs = []costFactory{func() mage.Cost { return mage.MillCost(amount) }}
	} else if cost.Name == "PayManaX" {
		mana, dynamic, err = compileResolutionManaCost(cost, ctx)
	} else if cost.Name == "PayLife" && len(cost.Args) == 1 && cost.Args[0].Name == "ManaValueOfTheCardRevealedThisWay" {
		if err := arity(cost, 1); err != nil {
			return nil, err
		}
		life, err = compileRevealedManaValue(cost.Args[0], ctx)
	} else if cost.Name == "Or" || cost.Name == "PayLife" {
		var build costFactory
		build, err = compileFixedResolutionAlternative(cost)
		costs = []costFactory{build}
	} else if cost.Name != "PayMana" && cost.Name != "SacrificeAPermanent" && cost.Name != "SacrificeNumberPermanents" && cost.Name != "DiscardACard" {
		return nil, unsupported(cost, "optional payment cost")
	} else {
		costs, _, err = compileCosts(cost)
	}
	if err != nil {
		return nil, err
	}
	if continuation.Name != "Unless" && continuation.Name != "If" {
		return nil, problem(continuation, "payment requires an adjacent payment continuation")
	}
	if err := arity(continuation, 2); err != nil {
		return nil, err
	}
	condition := continuation.Args[0]
	if condition.Kind != Symbol || (condition.Name != "CostWasPaid" && condition.Name != "CostWasntPaid") {
		return nil, unsupported(condition, "payment condition")
	}
	parts, err := list(continuation.Args[1])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(continuation, "payment continuation requires at least one action")
	}
	body, err := compileActionSequence(parts, ctx)
	if err != nil {
		return nil, err
	}
	// Keep the payment and its result together so other payments cannot change it.
	return func(bindings instanceBindings) mage.Effect {
		values := make([]mage.Cost, len(costs))
		for i, build := range costs {
			values[i] = build()
		}
		effects := make([]mage.Effect, len(body))
		for i, build := range body {
			effects[i] = build(bindings)
		}
		if payment.Name == "MustCost" && sacrifice != nil {
			if (continuation.Name == "If") == (condition.Name == "CostWasPaid") {
				return mage.IfRequiredSacrificePaid(player.build(bindings), sacrifice.build(bindings), mage.Sequence(effects...))
			}
			return mage.UnlessRequiredSacrificePaid(player.build(bindings), sacrifice.build(bindings), mage.Sequence(effects...))
		}
		if payment.Name == "MustCost" {
			if (continuation.Name == "If") == (condition.Name == "CostWasPaid") {
				return mage.IfRequiredPaid(player.build(bindings), mage.Costs(values...), mage.Sequence(effects...))
			}
			return mage.UnlessRequiredPaid(player.build(bindings), mage.Costs(values...), mage.Sequence(effects...))
		}
		if sacrifice != nil {
			if (continuation.Name == "If") == (condition.Name == "CostWasPaid") {
				return mage.IfSacrificePaid(player.build(bindings), sacrifice.build(bindings), mage.Sequence(effects...))
			}
			return mage.UnlessSacrificePaid(player.build(bindings), sacrifice.build(bindings), mage.Sequence(effects...))
		}
		if dynamic != nil {
			if (continuation.Name == "If") == (condition.Name == "CostWasPaid") {
				return mage.IfManaPaid(player.build(bindings), mana, dynamic(bindings), mage.Sequence(effects...))
			}
			return mage.UnlessManaPaid(player.build(bindings), mana, dynamic(bindings), mage.Sequence(effects...))
		}
		if life != nil {
			if (continuation.Name == "If") == (condition.Name == "CostWasPaid") {
				return mage.IfLifePaid(player.build(bindings), life(bindings), mage.Sequence(effects...))
			}
			return mage.UnlessLifePaid(player.build(bindings), life(bindings), mage.Sequence(effects...))
		}
		if (continuation.Name == "If") == (condition.Name == "CostWasPaid") {
			return mage.IfPaid(player.build(bindings), mage.Costs(values...), mage.Sequence(effects...))
		}
		return mage.UnlessPaid(player.build(bindings), mage.Costs(values...), mage.Sequence(effects...))
	}, nil
}

func compileFixedResolutionAlternative(n Node) (costFactory, error) {
	if n.Name != "Or" {
		if n.Name != "PayMana" && n.Name != "PayLife" {
			return nil, unsupported(n, "fixed resolution alternative")
		}
		costs, hasX, err := compileCosts(n)
		if err != nil {
			return nil, err
		}
		if hasX {
			return nil, problem(n, "resolution alternatives require fixed costs")
		}
		return costs[0], nil
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	parts, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) < 2 {
		return nil, problem(n, "supply at least two resolution cost alternatives")
	}
	options := make([]costFactory, len(parts))
	for i, part := range parts {
		options[i], err = compileFixedResolutionAlternative(part)
		if err != nil {
			return nil, err
		}
	}
	return func() mage.Cost {
		costs := make([]mage.Cost, len(options))
		for i, build := range options {
			costs[i] = build()
		}
		return mage.EitherCost(costs...)
	}, nil
}

func compileResolutionManaCost(n Node, ctx scope) (core.ManaCost, valueFactory, error) {
	if err := arity(n, 2); err != nil {
		return core.ManaCost{}, nil, err
	}
	text, err := manaCost(n.Args[0])
	if err != nil {
		return core.ManaCost{}, nil, err
	}
	cost := core.ParseManaCost(text)
	if !cost.HasX {
		return core.ManaCost{}, nil, problem(n, "PayManaX requires an X mana symbol")
	}
	amount := n.Args[1]
	if amount.Name == "ManaValueOfSpell" {
		if err := arity(amount, 1); err != nil {
			return core.ManaCost{}, nil, err
		}
		spell, err := compileSpellRef(amount.Args[0], ctx)
		if err != nil {
			return core.ManaCost{}, nil, err
		}
		return cost, func(bindings instanceBindings) mage.ResolutionValue {
			return mage.ManaValueOfSpellWithLKI(spell.build(bindings))
		}, nil
	}
	if amount.Name == "ValueX" && ctx.createdValue != nil {
		value, err := compileInstanceValue(amount, ctx)
		return cost, value, err
	}
	value, err := compileValue(amount, ctx)
	if err != nil {
		return core.ManaCost{}, nil, err
	}
	if amount.Name == "Integer" && amount.Args[0].Integer < 0 {
		return core.ManaCost{}, nil, problem(amount, "mana payment requires a nonnegative integer")
	}
	return cost, func(instanceBindings) mage.ResolutionValue { return mage.ValueAtResolution(value) }, nil
}
