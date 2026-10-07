package ron

import "github.com/benprew/mage-go/pkg/mage"

func compilePlayerUntapLimit(n Node) (optionFactory, error) {
	return compileConditionalPlayerUntapLimit(n, nil)
}

func compileConditionalPlayerUntapLimit(n Node, condition mage.StateCondition) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if n.Args[0].Kind != Symbol || n.Args[0].Name != "AnyPlayer" {
		return nil, unsupported(n.Args[0], "untap limit player filter")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 {
		return nil, problem(n.Args[1], "supply one untap limit operation")
	}
	operation := operations[0]
	if operation.Name != "CantUntapMoreThanNumberPermanentsDuringTheirUntapStep" {
		return nil, unsupported(operation, "all-player rule operation")
	}
	if err := arity(operation, 2); err != nil {
		return nil, err
	}
	value := operation.Args[0]
	if value.Name != "Integer" {
		return nil, unsupported(value, "fixed untap limit")
	}
	if err := arity(value, 1); err != nil {
		return nil, err
	}
	if value.Args[0].Kind != IntegerNode || value.Args[0].Integer < 0 {
		return nil, problem(value, "supply a nonnegative integer untap limit")
	}
	filter, err := compileFilter(operation.Args[1])
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		rule := mage.LimitUntapsDuringStep(filter, value.Args[0].Integer)
		if condition != nil {
			rule = mage.LimitUntapsDuringStepIf(filter, value.Args[0].Integer, condition)
		}
		return mage.WithAbility(mage.Static(rule))
	}, nil
}
