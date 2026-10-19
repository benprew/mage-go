package ron

import "github.com/benprew/mage-go/pkg/mage"

type damageLifeFloorPlan struct {
	floor int
}

func (p damageLifeFloorPlan) build() mage.CardOption {
	return mage.WithAbility(mage.Static(mage.LimitDamageLifeLoss(mage.You, p.floor)))
}

func compileDamageLifeFloor(operation Node) (optionFactory, error) {
	plan, err := compileDamageLifeFloorPlan(operation)
	if err != nil {
		return nil, err
	}
	return plan.build, nil
}

func compileDamageLifeFloorPlan(operation Node) (*damageLifeFloorPlan, error) {
	if operation.Name != "DamageThatWouldReduceLifeTotalToLessThanNumberReducesItToThatNumberInstead" {
		return nil, unsupported(operation, "damage life-floor operation")
	}
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
	return &damageLifeFloorPlan{floor: floor}, nil
}
