package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileStaticSpellCostIncrease(n Node) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	player, err := symbol(n.Args[0])
	if err != nil {
		return nil, err
	}
	controllerOnly := n.Name == "PlayerEffect"
	if (controllerOnly && player != "You") || (!controllerOnly && player != "AnyPlayer") {
		return nil, unsupported(n.Args[0], "spell cost increase player")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 {
		return nil, problem(n.Args[1], "supply one spell cost increase")
	}
	operation := operations[0]
	if operation.Name != "IncreaseSpellCost" {
		return nil, unsupported(operation, "spell cost increase")
	}
	if err := arity(operation, 2); err != nil {
		return nil, err
	}
	filter, err := compileCostSpellFilter(operation.Args[0])
	if err != nil {
		return nil, err
	}
	payment := operation.Args[1]
	if payment.Name != "PayMana" {
		return nil, unsupported(payment, "spell cost increase payment")
	}
	if err := arity(payment, 1); err != nil {
		return nil, err
	}
	parts, err := list(payment.Args[0])
	if err != nil {
		return nil, err
	}
	for _, part := range parts {
		if part.Name == "ManaCostX" {
			return nil, unsupported(part, "fixed spell cost increase")
		}
	}
	cost, err := manaCost(payment.Args[0])
	if err != nil {
		return nil, err
	}
	increase := core.ParseManaCost(cost)
	return func() mage.CardOption {
		return mage.WithStaticAbility(mage.IncreaseSpellCostStatic(filter, increase, controllerOnly))
	}, nil
}
