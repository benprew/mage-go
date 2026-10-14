package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileMatchingPermanentExileFilter(n Node) (mage.PermanentFilter, error) {
	hasController := false
	filter, err := compileMatchingTapFilter(n, &hasController)
	if err != nil {
		return mage.PermanentFilter{}, err
	}
	if !hasController {
		return mage.PermanentFilter{}, problem(n, "permanent exile payment requires your control restriction")
	}
	return filter, nil
}

func compileMatchingPermanentExileCost(n Node) (costFactory, error) {
	plan, err := compileExileCost(n)
	if err != nil {
		return nil, err
	}
	if plan == nil || plan.permanentFilter == nil {
		return nil, unsupported(n, "matching permanent exile casting cost")
	}
	return func() mage.Cost {
		return mage.ExileMatchingPermanentCost(*plan.permanentFilter, "Exile a matching permanent you control")
	}, nil
}
