package ron

import "github.com/benprew/mage-go/pkg/mage"

type exilePaymentPlan struct{ filter mage.CardFilter }

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
	if err := arity(object, 1); err != nil {
		return nil, err
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
		return compileExileCost(cost)
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
	if n.Name == "SacrificePermanent" && ctx.sourceCardPayment != nil {
		if _, _, err := compileCosts(n); err != nil {
			return nil, false, err
		}
		ctx.sourceCardPayment.producers++
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
		return []boundCostFactory{func(bindings instanceBindings) mage.Cost {
			return mage.ExileMatchingCardFromGraveyardCostInto(plan.filter, "Exile a matching card from your graveyard", bindings.exiles[ctx.exilePayment])
		}}, false, nil
	}
	if n.Name == "SacrificeAPermanent" && ctx.sacrificeValue == nil {
		cost, err := compileRequiredSacrificeCost(n, ctx)
		if err != nil {
			return nil, false, err
		}
		return []boundCostFactory{func(instanceBindings) mage.Cost { return cost() }}, false, nil
	}
	return compileActivationCosts(n, ctx.sacrificeValue, ctx.counterPayment)
}

func compileExilePaymentValue(n Node, ctx scope) (valueFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	ref := n.Args[0]
	if ref.Kind != Symbol || ref.Name != "TheCardExiledThisWay" || ctx.exilePayment == nil {
		return nil, problem(ref, "unbound or incompatible exile payment result")
	}
	return func(bindings instanceBindings) mage.ResolutionValue {
		return mage.FromNumber(bindings.exiles[ctx.exilePayment])
	}, nil
}
