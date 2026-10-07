package ron

import "github.com/benprew/mage-go/pkg/mage"

type sacrificeValuePlan struct {
	name string
}

type sacrificeValueBindings struct {
	power, value *mage.NumberBinding
}

type boundCostFactory func(instanceBindings) mage.Cost

type valueFactory func(instanceBindings) mage.ResolutionValue

func compileInstanceValue(n Node, ctx scope) (valueFactory, error) {
	if n.Name == "TheTotalNumberOfCountersOfTypeAmongPermanents" {
		return compileCounterInformation(n, ctx)
	}
	if n.Name == "ToughnessOfDeadPermanent" {
		if n.Kind != Symbol || ctx.hostDeath == nil {
			return nil, problem(n, "unbound or incompatible death event value")
		}
		return func(bindings instanceBindings) mage.ResolutionValue {
			return mage.FromNumber(bindings.hostDeaths[ctx.hostDeath].toughness)
		}, nil
	}
	if n.Name == "ManaValueOfExiled" {
		return compileExilePaymentValue(n, ctx)
	}
	if n.Name == "HalfRoundedDown" || n.Name == "HalfRoundedUp" {
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		value, err := compileInstanceValue(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.ResolutionValue {
			if n.Name == "HalfRoundedDown" {
				return mage.DivideRoundDown(value(bindings), mage.Fixed(2))
			}
			return mage.DivideRoundUp(value(bindings), mage.Fixed(2))
		}, nil
	}
	if n.Name == "Trigger_AmountOfDamageDealt" {
		if n.Kind != Symbol || ctx.damageAmount == nil {
			return nil, problem(n, "unbound or incompatible damage event amount")
		}
		return func(bindings instanceBindings) mage.ResolutionValue {
			return mage.FromNumber(bindings.damageAmounts[ctx.damageAmount])
		}, nil
	}
	if n.Name == "TheNumberOfCardsOfTypeInPlayersHand" {
		return compileHandCount(n, ctx)
	}
	if n.Name == "TheNumberOfPermanentsOnTheBattlefield" {
		return compileBattlefieldCount(n, ctx)
	}
	if n.Name == "Twice" {
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		value, err := compileInstanceValue(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.ResolutionValue {
			return mage.MultiplyValues(mage.Fixed(2), value(bindings))
		}, nil
	}
	if n.Name == "PowerOfPermanent" || n.Name == "ManaValueOfPermanent" {
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		permanent, err := compilePermanentRef(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.ResolutionValue {
			ref := permanent.build(bindings)
			if n.Name == "PowerOfPermanent" {
				return mage.PowerOfWithLKI(ref)
			}
			return mage.ManaValueOfWithLKI(ref)
		}, nil
	}
	if n.Name == "TheNumberOfCardsInPlayersHand" {
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		player, err := compilePlayerRef(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.ResolutionValue {
			return mage.HandSizeOf(player.build(bindings))
		}, nil
	}
	if n.Name == "Minus" {
		if err := arity(n, 2); err != nil {
			return nil, err
		}
		left, err := compileInstanceValue(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		right, err := compileInstanceValue(n.Args[1], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.ResolutionValue {
			return mage.SubtractValues(left(bindings), right(bindings))
		}, nil
	}
	if n.Name == "TheNumberOfCountersRemovedThisWay" {
		if n.Kind != Symbol || ctx.counterPayment == nil {
			return nil, problem(n, "unbound or incompatible counter payment value")
		}
		return func(bindings instanceBindings) mage.ResolutionValue {
			return mage.FromNumber(bindings.counters[ctx.counterPayment])
		}, nil
	}
	if n.Name == "ToughnessOfTheSacrificedCreature" || n.Name == "ManaValueOfTheSacrificedPermanent" {
		if n.Kind != Symbol || ctx.sacrificeValue == nil || ctx.sacrificeValue.name != n.Name {
			return nil, problem(n, "unbound or incompatible sacrifice payment value")
		}
		return func(bindings instanceBindings) mage.ResolutionValue {
			return mage.FromNumber(bindings.sacrifices[ctx.sacrificeValue].value)
		}, nil
	}
	value, err := compileValue(n, ctx)
	if err != nil {
		return nil, err
	}
	return func(instanceBindings) mage.ResolutionValue { return mage.ValueAtResolution(value) }, nil
}

func compileSacrificeValuePlan(cost, actions Node) (*sacrificeValuePlan, error) {
	var name string
	var scan func(Node) error
	scan = func(n Node) error {
		if n.Name == "ToughnessOfTheSacrificedCreature" || n.Name == "ManaValueOfTheSacrificedPermanent" {
			if name != "" && name != n.Name {
				return problem(n, "multiple sacrifice characteristic results are not supported")
			}
			name = n.Name
		}
		for _, arg := range n.Args {
			if err := scan(arg); err != nil {
				return err
			}
		}
		return nil
	}
	if err := scan(actions); err != nil {
		return nil, err
	}
	if name == "" {
		return nil, nil
	}
	var sacrifices []Node
	sourceSacrifice := false
	var find func(Node)
	find = func(n Node) {
		if n.Name == "SacrificePermanent" {
			sourceSacrifice = true
		}
		if n.Name == "SacrificeAPermanent" {
			sacrifices = append(sacrifices, n)
		}
		if n.Name == "And" && len(n.Args) == 1 {
			for _, child := range n.Args[0].Args {
				find(child)
			}
		}
	}
	find(cost)
	if len(sacrifices) != 1 || sourceSacrifice {
		return nil, problem(cost, "sacrifice value requires exactly one matching sacrifice cost")
	}
	if name == "ToughnessOfTheSacrificedCreature" {
		n := sacrifices[0]
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		f := n.Args[0]
		if f.Name != "IsCardtype" || len(f.Args) != 1 || f.Args[0].Kind != Symbol || f.Args[0].Name != "Creature" {
			return nil, unsupported(f, "creature sacrifice result filter")
		}
	}
	return &sacrificeValuePlan{name: name}, nil
}

func compileBoundCosts(n Node, paid *sacrificeValuePlan) ([]boundCostFactory, bool, error) {
	if paid == nil {
		costs, x, err := compileCosts(n)
		if err != nil {
			return nil, false, err
		}
		var out []boundCostFactory
		for _, build := range costs {
			out = append(out, func(instanceBindings) mage.Cost { return build() })
		}
		return out, x, nil
	}
	if n.Name == "And" {
		if err := arity(n, 1); err != nil {
			return nil, false, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return nil, false, err
		}
		var out []boundCostFactory
		var x bool
		for _, part := range parts {
			costs, hasX, err := compileBoundCosts(part, paid)
			if err != nil {
				return nil, false, err
			}
			out = append(out, costs...)
			x = x || hasX
		}
		return out, x, nil
	}
	if n.Name != "SacrificeAPermanent" {
		return compileBoundCosts(n, nil)
	}
	if err := arity(n, 1); err != nil {
		return nil, false, err
	}
	filter, err := compileSacrificeCostFilter(n.Args[0])
	if err != nil {
		return nil, false, err
	}
	return []boundCostFactory{func(bindings instanceBindings) mage.Cost {
		saved := bindings.sacrifices[paid]
		if paid.name == "ToughnessOfTheSacrificedCreature" {
			return mage.SacrificeCreatureIncludingSourceCostWithStats(saved.power, saved.value)
		}
		return mage.SacrificeMatchingIncludingSourceCostWithManaValue(filter, "Sacrifice a matching permanent", saved.value)
	}}, false, nil
}
