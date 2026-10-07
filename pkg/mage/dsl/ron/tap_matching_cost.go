package ron

import (
	"fmt"

	"github.com/benprew/mage-go/pkg/mage"
)

func compileMatchingTapCost(n Node) (costFactory, error) {
	count := 1
	var filterNode Node
	if n.Name == "TapAPermanent" {
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		filterNode = n.Args[0]
	} else {
		if err := arity(n, 2); err != nil {
			return nil, err
		}
		amount := n.Args[0]
		if amount.Name != "Integer" {
			return nil, unsupported(amount, "fixed tap count")
		}
		if err := arity(amount, 1); err != nil {
			return nil, err
		}
		if amount.Args[0].Kind != IntegerNode || amount.Args[0].Integer <= 0 {
			return nil, problem(amount, "tap count requires a positive integer literal")
		}
		count = amount.Args[0].Integer
		filterNode = n.Args[1]
	}
	hasController := false
	filter, err := compileMatchingTapFilter(filterNode, &hasController)
	if err != nil {
		return nil, err
	}
	return func() mage.Cost {
		return mage.TapNMatchingIncludingSourceCost(count, filter, fmt.Sprintf("Tap %d matching untapped permanents you control", count))
	}, nil
}

func compileMatchingTapFilter(n Node, hasController *bool) (mage.PermanentFilter, error) {
	if n.Name == "ControlledByAPlayer" {
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, err
		}
		player := n.Args[0]
		if player.Name != "SinglePlayer" {
			return mage.PermanentFilter{}, unsupported(player, "tap cost controller")
		}
		if err := arity(player, 1); err != nil {
			return mage.PermanentFilter{}, err
		}
		if player.Args[0].Kind != Symbol || player.Args[0].Name != "You" {
			return mage.PermanentFilter{}, unsupported(player.Args[0], "tap cost controller")
		}
		if *hasController {
			return mage.PermanentFilter{}, problem(n, "supply one tap cost controller restriction")
		}
		*hasController = true
		return mage.AnyPermanent, nil
	}
	if n.Name == "And" {
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return mage.PermanentFilter{}, err
		}
		if len(parts) == 0 {
			return mage.PermanentFilter{}, problem(n, "tap cost requires a nonempty filter")
		}
		filters := make([]mage.PermanentFilter, len(parts))
		for i, part := range parts {
			filters[i], err = compileMatchingTapFilter(part, hasController)
			if err != nil {
				return mage.PermanentFilter{}, err
			}
		}
		return mage.And(filters...), nil
	}
	return compileSacrificeCostFilter(n)
}
