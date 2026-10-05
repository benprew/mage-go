package ron

import (
	"fmt"

	"github.com/benprew/mage-go/pkg/mage"
)

func compileSacrificeCostFilter(n Node) (mage.PermanentFilter, error) {
	if n.Name != "And" && n.Name != "Or" {
		return compileFilter(n)
	}
	if err := arity(n, 1); err != nil {
		return mage.PermanentFilter{}, err
	}
	parts, err := list(n.Args[0])
	if err != nil {
		return mage.PermanentFilter{}, err
	}
	if len(parts) == 0 {
		return mage.PermanentFilter{}, problem(n, "sacrifice cost requires a nonempty filter")
	}
	filters := make([]mage.PermanentFilter, len(parts))
	for i, part := range parts {
		filters[i], err = compileSacrificeCostFilter(part)
		if err != nil {
			return mage.PermanentFilter{}, err
		}
	}
	if n.Name == "And" {
		return mage.And(filters...), nil
	}
	return mage.Or(filters...), nil
}

func compileSacrificeNumberCost(n Node) (costFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	count := n.Args[0]
	if count.Name != "Integer" {
		return nil, unsupported(count, "fixed sacrifice count")
	}
	if err := arity(count, 1); err != nil {
		return nil, err
	}
	if count.Args[0].Kind != IntegerNode || count.Args[0].Integer <= 0 {
		return nil, problem(count, "sacrifice count requires a positive integer literal")
	}
	filter, err := compileSacrificeCostFilter(n.Args[1])
	if err != nil {
		return nil, err
	}
	amount := count.Args[0].Integer
	return func() mage.Cost {
		return mage.SacrificeNMatchingIncludingSourceCost(amount, filter, fmt.Sprintf("Sacrifice %d matching permanents", amount))
	}, nil
}
