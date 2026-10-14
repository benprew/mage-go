package ron

import "github.com/benprew/mage-go/pkg/mage"

type characteristicController uint8

const (
	characteristicAll characteristicController = iota
	characteristicYou
	characteristicOpponent
)

type characteristicFilterPlan struct {
	filter     mage.PermanentFilter
	controller characteristicController
}

func compileAdditiveCharacteristicCount(n Node) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	offset := n.Args[0]
	if offset.Name != "Integer" {
		return nil, unsupported(offset, "additive characteristic offset")
	}
	if err := arity(offset, 1); err != nil {
		return nil, err
	}
	if offset.Args[0].Kind != IntegerNode {
		return nil, problem(offset.Args[0], "characteristic offset requires an integer")
	}
	value := n.Args[1]
	if value.Name != "TheNumberOfPermanentsOnTheBattlefield" {
		return nil, unsupported(value, "additive characteristic count")
	}
	if err := arity(value, 1); err != nil {
		return nil, err
	}
	if err := validateCharacteristicCountFilter(value.Args[0]); err != nil {
		return nil, err
	}
	plan, err := compileAdditiveCharacteristicFilter(value.Args[0])
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		var effect mage.ContinuousEffect
		switch plan.controller {
		case characteristicYou:
			effect = mage.PTEqualsControlledCountPlus(plan.filter, offset.Args[0].Integer)
		case characteristicOpponent:
			effect = mage.PTEqualsOpponentCountPlus(plan.filter, offset.Args[0].Integer)
		default:
			effect = mage.PTEqualsCountPlus(plan.filter, offset.Args[0].Integer)
		}
		return mage.WithStaticAbility(effect)
	}, nil
}

func compileAdditiveCharacteristicFilter(n Node) (characteristicFilterPlan, error) {
	if n.Name == "ControlledByAPlayer" {
		if err := arity(n, 1); err != nil {
			return characteristicFilterPlan{}, err
		}
		if n.Args[0].Kind == Symbol && n.Args[0].Name == "Opponent" {
			return characteristicFilterPlan{filter: mage.AnyPermanent, controller: characteristicOpponent}, nil
		}
		plan, err := compileStaticPermanentFilter(n)
		return characteristicFilterPlan{filter: plan.filter, controller: characteristicYou}, err
	}
	if n.Name != "And" && n.Name != "Or" {
		plan, err := compileStaticPermanentFilter(n)
		return characteristicFilterPlan{filter: plan.filter}, err
	}
	parts := n.Args[0].Args
	filters := make([]mage.PermanentFilter, 0, len(parts))
	plan := characteristicFilterPlan{}
	for _, part := range parts {
		child, err := compileAdditiveCharacteristicFilter(part)
		if err != nil {
			return characteristicFilterPlan{}, err
		}
		if child.controller != characteristicAll {
			if n.Name == "Or" {
				return characteristicFilterPlan{}, unsupported(part, "characteristic controller disjunction")
			}
			if plan.controller != characteristicAll && plan.controller != child.controller {
				return characteristicFilterPlan{}, problem(part, "characteristic count has incompatible controller restrictions")
			}
			plan.controller = child.controller
		}
		filters = append(filters, child.filter)
	}
	plan.filter = mage.And(filters...)
	if n.Name == "Or" {
		plan.filter = mage.Or(filters...)
	}
	return plan, nil
}
