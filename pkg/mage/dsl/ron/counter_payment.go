package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type counterPaymentPlan struct{ counter core.CounterType }

func compileCounterPaymentPlan(n Node) (*counterPaymentPlan, error) {
	var removals []Node
	var walk func(Node) error
	walk = func(n Node) error {
		switch n.Name {
		case "And":
			if err := arity(n, 1); err != nil {
				return err
			}
			parts, err := list(n.Args[0])
			if err != nil {
				return err
			}
			for _, part := range parts {
				if err := walk(part); err != nil {
					return err
				}
			}
		case "RemoveCounters":
			removals = append(removals, n)
		}
		return nil
	}
	if err := walk(n); err != nil {
		return nil, err
	}
	var plan *counterPaymentPlan
	for _, removal := range removals {
		if err := arity(removal, 1); err != nil {
			return nil, err
		}
		parts, err := list(removal.Args[0])
		if err != nil {
			return nil, err
		}
		for _, instruction := range parts {
			if instruction.Name != "AnyNumberOfCountersOfTypeFromPermanent" {
				continue
			}
			if len(removals) != 1 || len(parts) != 1 {
				return nil, problem(removal, "chosen counter payment requires one counter removal cost")
			}
			if err := arity(instruction, 2); err != nil {
				return nil, err
			}
			counter, err := compilePermanentCounterType(instruction.Args[0])
			if err != nil {
				return nil, err
			}
			source := instruction.Args[1]
			if source.Kind != Symbol || source.Name != "ThisPermanent" {
				return nil, unsupported(source, "counter cost source")
			}
			plan = &counterPaymentPlan{counter: counter}
		}
	}
	return plan, nil
}

func compileActivationCosts(n Node, sacrifice *sacrificeValuePlan, counters *counterPaymentPlan) ([]boundCostFactory, bool, error) {
	if counters == nil {
		return compileBoundCosts(n, sacrifice)
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
			next, hasX, err := compileActivationCosts(part, sacrifice, counters)
			if err != nil {
				return nil, false, err
			}
			costs = append(costs, next...)
			x = x || hasX
		}
		return costs, x, nil
	}
	if n.Name == "RemoveCounters" {
		return []boundCostFactory{func(bindings instanceBindings) mage.Cost {
			return mage.RemoveAnyCountersCost(counters.counter, bindings.counters[counters])
		}}, false, nil
	}
	return compileBoundCosts(n, sacrifice)
}

func compileBoundRepeatedMana(n Node, ctx scope) ([]effectFactory, error) {
	value, err := compileInstanceValue(n.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	productions, err := compileMana(n.Args[0])
	if err != nil {
		return nil, err
	}
	var effects []effectFactory
	for _, production := range productions {
		color := production.Color
		effects = append(effects, func(bindings instanceBindings) mage.Effect {
			if color == core.AnyColor {
				return mage.AddManaOfAnyOneColor(value(bindings))
			}
			return mage.AddManaAmount(color, value(bindings))
		})
	}
	return effects, nil
}
