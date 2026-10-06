package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileActionSequence(parts []Node, ctx scope) ([]effectFactory, error) {
	var effects []effectFactory
	for i := 0; i < len(parts); i++ {
		part := parts[i]
		if part.Name == "EachPlayerAction" && part.Kind == Tuple && len(part.Args) == 2 && part.Args[1].Name == "ShuffleHandIntoLibrary" {
			if i+1 == len(parts) {
				return nil, problem(part, "hand shuffle requires an adjacent all-player draw of its result")
			}
			effect, err := compileHandShuffleDraw(part, parts[i+1])
			if err != nil {
				return nil, err
			}
			effects = append(effects, effect)
			i++
			continue
		}
		if part.Name == "PlayerMayCost" || part.Name == "MayCost" || part.Name == "MustCost" {
			if i+1 == len(parts) {
				return nil, problem(part, "payment requires an adjacent payment continuation")
			}
			effect, err := compileResolutionPayment(part, parts[i+1], ctx)
			if err != nil {
				return nil, err
			}
			effects = append(effects, effect)
			i++
			continue
		}
		compiled, err := compileEffect(part, ctx)
		if err != nil {
			return nil, err
		}
		effects = append(effects, compiled...)
	}
	return effects, nil
}

func compileResolutionPayment(payment, continuation Node, ctx scope) (effectFactory, error) {
	var player playerReferencePlan
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
	}
	var dynamic valueFactory
	var mana core.ManaCost
	var costs []costFactory
	var err error
	if payment.Name == "MustCost" {
		var build costFactory
		build, err = compileRequiredSacrificeCost(cost, ctx)
		costs = []costFactory{build}
	} else if cost.Name == "MillNumberCards" {
		var amount int
		amount, err = compileMillCount(cost, true)
		costs = []costFactory{func() mage.Cost { return mage.MillCost(amount) }}
	} else if cost.Name == "PayManaX" {
		mana, dynamic, err = compileResolutionManaCost(cost, ctx)
	} else if cost.Name != "PayMana" && cost.Name != "SacrificeAPermanent" && cost.Name != "SacrificeNumberPermanents" {
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
		if payment.Name == "MustCost" {
			if (continuation.Name == "If") == (condition.Name == "CostWasPaid") {
				return mage.IfRequiredPaid(player.build(bindings), mage.Costs(values...), mage.Sequence(effects...))
			}
			return mage.UnlessRequiredPaid(player.build(bindings), mage.Costs(values...), mage.Sequence(effects...))
		}
		if dynamic != nil {
			if (continuation.Name == "If") == (condition.Name == "CostWasPaid") {
				return mage.IfManaPaid(player.build(bindings), mana, dynamic(bindings), mage.Sequence(effects...))
			}
			return mage.UnlessManaPaid(player.build(bindings), mana, dynamic(bindings), mage.Sequence(effects...))
		}
		if (continuation.Name == "If") == (condition.Name == "CostWasPaid") {
			return mage.IfPaid(player.build(bindings), mage.Costs(values...), mage.Sequence(effects...))
		}
		return mage.UnlessPaid(player.build(bindings), mage.Costs(values...), mage.Sequence(effects...))
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
	value, err := compileValue(amount, ctx)
	if err != nil {
		return core.ManaCost{}, nil, err
	}
	if amount.Name == "Integer" && amount.Args[0].Integer < 0 {
		return core.ManaCost{}, nil, problem(amount, "mana payment requires a nonnegative integer")
	}
	return cost, func(instanceBindings) mage.ResolutionValue { return mage.ValueAtResolution(value) }, nil
}
