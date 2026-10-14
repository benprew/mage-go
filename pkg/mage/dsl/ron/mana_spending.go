package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func isManaSpendingRule(n Node) bool {
	if len(n.Args) != 2 || n.Args[1].Kind != List {
		return false
	}
	for _, operation := range n.Args[1].Args {
		if operation.Name == "MaySpendColoredManaAsThoughItWereAnotherColor" {
			return true
		}
	}
	return false
}

func compileManaSpendingRule(n Node) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	player, err := symbol(n.Args[0])
	if err != nil {
		return nil, err
	}
	if player != "You" {
		return nil, unsupported(n.Args[0], "mana spending player")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 {
		return nil, problem(n.Args[1], "supply one mana spending operation")
	}
	operation := operations[0]
	if err := arity(operation, 2); err != nil {
		return nil, err
	}
	from, err := color(operation.Args[0])
	if err != nil {
		return nil, err
	}
	to, err := color(operation.Args[1])
	if err != nil {
		return nil, err
	}
	if from == core.Colorless || to == core.Colorless {
		return nil, problem(operation, "supply two mana colors")
	}
	return func() mage.CardOption { return mage.WithStaticAbility(mage.ControllerManaSpending(from, to)) }, nil
}
