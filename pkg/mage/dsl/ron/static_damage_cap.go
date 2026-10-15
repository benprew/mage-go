package ron

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileStaticDamageCap(n Node) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	event := n.Args[0]
	if event.Name != "AnAmountOfDamageWouldBeDealtByASourceToRecipient" {
		return nil, unsupported(event, "static damage amount event")
	}
	if err := arity(event, 3); err != nil {
		return nil, err
	}
	comparison := event.Args[0]
	if comparison.Name != "GreaterThanOrEqualTo" {
		return nil, unsupported(comparison, "static damage threshold")
	}
	if err := arity(comparison, 1); err != nil {
		return nil, err
	}
	threshold, err := fixedDamageCapAmount(comparison.Args[0])
	if err != nil {
		return nil, err
	}
	kinds, err := compileDamageCapSourceTypes(event.Args[1])
	if err != nil {
		return nil, err
	}
	recipient := event.Args[2]
	if recipient.Name != "Player" {
		return nil, unsupported(recipient, "static damage amount recipient")
	}
	if err := arity(recipient, 1); err != nil {
		return nil, err
	}
	if recipient.Args[0].Kind != Symbol || recipient.Args[0].Name != "You" {
		return nil, unsupported(recipient.Args[0], "static damage amount player")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 || operations[0].Name != "DealDamageInstead" {
		return nil, problem(n.Args[1], "supply one DealDamageInstead operation")
	}
	if err := arity(operations[0], 1); err != nil {
		return nil, err
	}
	amount, err := fixedDamageCapAmount(operations[0].Args[0])
	if err != nil {
		return nil, err
	}
	if threshold <= 0 || amount >= threshold {
		return nil, problem(n, "supply a positive threshold and a smaller damage amount")
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.Static(mage.LimitPlayerDamageFromTypes(mage.You, threshold, amount, kinds...)))
	}, nil
}

func fixedDamageCapAmount(n Node) (int, error) {
	if n.Name != "Integer" {
		return 0, unsupported(n, "fixed damage amount")
	}
	if err := arity(n, 1); err != nil {
		return 0, err
	}
	if n.Args[0].Kind != IntegerNode || n.Args[0].Integer < 0 {
		return 0, problem(n, "supply a nonnegative integer damage amount")
	}
	return n.Args[0].Integer, nil
}

func compileDamageCapSourceTypes(n Node) ([]core.CardType, error) {
	filters := []Node{n}
	if n.Name == "Or" {
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		var err error
		filters, err = list(n.Args[0])
		if err != nil {
			return nil, err
		}
		if len(filters) < 2 {
			return nil, problem(n, "supply at least two source types in Or")
		}
	}
	kinds := make([]core.CardType, 0, len(filters))
	for _, filter := range filters {
		if filter.Name != "IsCardtype" {
			return nil, unsupported(filter, "damage amount source filter")
		}
		if err := arity(filter, 1); err != nil {
			return nil, err
		}
		kind, err := cardType(filter.Args[0])
		if err != nil {
			return nil, err
		}
		if slices.Contains(kinds, kind) {
			return nil, problem(filter, "duplicate damage source type")
		}
		kinds = append(kinds, kind)
	}
	return kinds, nil
}
