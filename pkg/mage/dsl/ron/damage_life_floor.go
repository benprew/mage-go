package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileDamageLifeFloor(operation Node) (optionFactory, error) {
	if err := arity(operation, 1); err != nil {
		return nil, err
	}
	floor, err := fixedDamageCapAmount(operation.Args[0])
	if err != nil {
		return nil, err
	}
	if floor <= 0 {
		return nil, problem(operation, "supply a positive life total")
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.Static(mage.LimitDamageLifeLoss(mage.You, floor)))
	}, nil
}
