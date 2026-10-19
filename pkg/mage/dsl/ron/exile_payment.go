package ron

import "github.com/benprew/mage-go/pkg/mage"

type exilePaymentPlan struct {
	permanentFilter *mage.PermanentFilter
	filter          mage.CardFilter
	singleGraveyard bool
	amount          int
}

func compileExileCost(n Node) (*exilePaymentPlan, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	objects, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(objects) != 1 || len(operations) != 0 {
		return nil, problem(n, "exile payment requires one object and no operations")
	}
	object := objects[0]
	if object.Name == "NumberCardsInASingleGraveyard" {
		return compileSingleGraveyardExileCost(object)
	}
	if err := arity(object, 1); err != nil {
		return nil, err
	}
	if object.Name == "APermanent" {
		filter, err := compileMatchingPermanentExileFilter(object.Args[0])
		if err != nil {
			return nil, err
		}
		return &exilePaymentPlan{permanentFilter: &filter}, nil
	}
	if object.Name == "Permanent" {
		if object.Args[0].Kind != Symbol || object.Args[0].Name != "ThisPermanent" {
			return nil, unsupported(object.Args[0], "exile payment source")
		}
		return nil, nil
	}
	if object.Name != "ACardInGraveyards" {
		return nil, unsupported(object, "exile payment object")
	}
	filter := object.Args[0]
	if filter.Name != "And" {
		return nil, unsupported(filter, "graveyard exile payment filter")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	parts, err := list(filter.Args[0])
	if err != nil {
		return nil, err
	}
	var filters []mage.CardFilter
	own := false
	for _, part := range parts {
		if part.Name != "InAPlayersGraveyard" {
			f, err := compileHandCardFilter(part)
			if err != nil {
				return nil, err
			}
			filters = append(filters, f)
			continue
		}
		if own {
			return nil, problem(part, "duplicate graveyard payment owner")
		}
		if err := arity(part, 1); err != nil {
			return nil, err
		}
		player := part.Args[0]
		if player.Name != "SinglePlayer" {
			return nil, unsupported(player, "graveyard payment player")
		}
		if err := arity(player, 1); err != nil {
			return nil, err
		}
		if player.Args[0].Kind != Symbol || player.Args[0].Name != "You" {
			return nil, unsupported(player.Args[0], "graveyard payment player")
		}
		own = true
	}
	if !own || len(filters) == 0 {
		return nil, problem(filter, "graveyard payment requires your graveyard and a card filter")
	}
	return &exilePaymentPlan{filter: mage.NewCardFilter("matching graveyard card", func(c mage.Card) bool {
		for _, filter := range filters {
			if !filter.Match(c) {
				return false
			}
		}
		return true
	})}, nil
}

func compileExilePaymentPlan(cost Node) (*exilePaymentPlan, error) {
	if cost.Name == "Exile" {
		plan, err := compileExileCost(cost)
		if plan != nil && plan.permanentFilter != nil {
			return nil, err
		}
		return plan, err
	}
	if cost.Name != "And" {
		return nil, nil
	}
	if err := arity(cost, 1); err != nil {
		return nil, err
	}
	parts, err := list(cost.Args[0])
	if err != nil {
		return nil, err
	}
	var plan *exilePaymentPlan
	for _, part := range parts {
		child, err := compileExilePaymentPlan(part)
		if err != nil {
			return nil, err
		}
		if child != nil {
			if plan != nil {
				return nil, problem(cost, "one graveyard exile payment is supported per activation")
			}
			plan = child
		}
	}
	return plan, nil
}

func compileExileActivationCosts(n Node, ctx scope) ([]boundCostFactory, bool, error) {
	if ctx.discardFacts != nil && (n.Name == "DiscardCard" || n.Name == "DiscardACardAtRandom") {
		ctx.discardFacts.producers++
	}
	if n.Name == "DiscardACard" && ctx.discardFacts != nil {
		if _, err := symbol(n); err != nil {
			return nil, false, err
		}
		ctx.discardFacts.producers++
		ctx.discardFacts.single = true
		return []boundCostFactory{func(bindings instanceBindings) mage.Cost {
			if len(ctx.discardFacts.filters) == 0 {
				return mage.DiscardCost(1)
			}
			facts := make([]mage.CardPaymentFact, len(ctx.discardFacts.filters))
			for i, filter := range ctx.discardFacts.filters {
				facts[i] = mage.CardPaymentFact{Filter: filter, Result: bindings.discardFacts[ctx.discardFacts][i]}
			}
			return mage.DiscardCardCostWithFacts(facts...)
		}}, false, nil
	}

	if n.Name == "PayManaCostOfPermanent" {
		return compilePermanentManaCost(n, ctx)
	}
	if n.Name == "And" {
		if err := arity(n, 1); err != nil {
			return nil, false, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return nil, false, err
		}
		var costs []boundCostFactory
		var x bool
		for _, part := range parts {
			next, hasX, err := compileExileActivationCosts(part, ctx)
			if err != nil {
				return nil, false, err
			}
			costs = append(costs, next...)
			x = x || hasX
		}
		return costs, x, nil
	}
	if n.Name == "TapPermanent" && len(n.Args) == 1 && n.Args[0].Name == "HostPermanent" {
		if err := arity(n, 1); err != nil {
			return nil, false, err
		}
		if _, err := compilePermanentRef(n.Args[0], ctx); err != nil {
			return nil, false, err
		}
		return []boundCostFactory{func(instanceBindings) mage.Cost { return mage.TapAttachedPermanentCost() }}, false, nil
	}
	if n.Name == "SacrificePermanent" && ctx.sourceCardPayment != nil {
		if _, _, err := compileCosts(n); err != nil {
			return nil, false, err
		}
		ctx.sourceCardPayment.producers++
		if ctx.pluralSacrifice != nil && ctx.pluralSacrifice.source {
			return []boundCostFactory{func(bindings instanceBindings) mage.Cost {
				saved := bindings.pluralSacrifices[ctx.pluralSacrifice]
				return mage.SacrificeSourceAndAnyMatchingCostInto(ctx.pluralSacrifice.filter, "Sacrifice the source and matching permanents you control", saved.power, saved.cards, bindings.sourceCards[ctx.sourceCardPayment])
			}}, false, nil
		}
		return []boundCostFactory{func(bindings instanceBindings) mage.Cost {
			if ctx.sourceCardPayment.used {
				return mage.SacrificeSourceCostWithCard(bindings.sourceCards[ctx.sourceCardPayment])
			}
			return mage.SacrificeSourceCost()
		}}, false, nil
	}
	if n.Name == "Exile" {
		plan, err := compileExileCost(n)
		if err != nil {
			return nil, false, err
		}
		if plan == nil {
			ctx.sourceCardPayment.producers++
			return []boundCostFactory{func(bindings instanceBindings) mage.Cost {
				if ctx.sourceCardPayment.used {
					return mage.ExileSourceCostWithCard(bindings.sourceCards[ctx.sourceCardPayment])
				}
				return mage.ExileSourceCost()
			}}, false, nil
		}
		if plan.permanentFilter != nil {
			return []boundCostFactory{func(instanceBindings) mage.Cost {
				return mage.ExileMatchingPermanentCost(*plan.permanentFilter, "Exile a matching permanent you control")
			}}, false, nil
		}
		if plan.singleGraveyard {
			return []boundCostFactory{func(instanceBindings) mage.Cost {
				return mage.ExileMatchingCardsFromSingleGraveyardCost(plan.amount, plan.filter, mage.EachPlayer)
			}}, false, nil
		}
		return []boundCostFactory{func(bindings instanceBindings) mage.Cost {
			return mage.ExileMatchingCardFromGraveyardCostInto(plan.filter, "Exile a matching card from your graveyard", bindings.exiles[ctx.exilePayment])
		}}, false, nil
	}
	if n.Name == "SacrificeAnyNumberOfPermanents" && ctx.pluralSacrifice != nil {
		if ctx.pluralSacrifice.source {
			return nil, false, nil
		}
		return []boundCostFactory{func(bindings instanceBindings) mage.Cost {
			saved := bindings.pluralSacrifices[ctx.pluralSacrifice]
			return mage.SacrificeAnyMatchingCostInto(ctx.pluralSacrifice.filter, "Sacrifice matching permanents you control", saved.power, saved.cards)
		}}, false, nil
	}
	if n.Name == "SacrificeAPermanent" && ctx.sacrificeValue == nil {
		cost, err := compileRequiredSacrificeCost(n, ctx)
		if err != nil {
			return nil, false, err
		}
		return []boundCostFactory{func(bindings instanceBindings) mage.Cost {
			if result := ctx.sacrificeSubtype; result != nil && result.subtype != "" {
				return mage.SacrificeMatchingIncludingSourceCostWithSubtype(result.filter, "Sacrifice a matching permanent", result.subtype, sacrificeSubtypeBinding(bindings, result))
			}
			return cost()
		}}, false, nil
	}
	return compileActivationCosts(n, ctx.sacrificeValue, ctx.counterPayment)
}

func compileExilePaymentValue(n Node, ctx scope) (valueFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	ref := n.Args[0]
	if ref.Kind != Symbol || ref.Name != "TheCardExiledThisWay" || ctx.exilePayment == nil || ctx.exilePayment.singleGraveyard || ctx.exilePayment.permanentFilter != nil {
		return nil, problem(ref, "unbound or incompatible exile payment result")
	}
	return func(bindings instanceBindings) mage.ResolutionValue {
		return mage.FromNumber(bindings.exiles[ctx.exilePayment])
	}, nil
}
