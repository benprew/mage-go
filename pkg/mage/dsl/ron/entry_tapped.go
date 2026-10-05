package ron

import (
	"math"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compilePermanentEntry(n Node) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	source := n.Args[0]
	if source.Kind != Symbol || source.Name != "ThisPermanent" {
		return nil, unsupported(source, "entry source")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 {
		return nil, problem(n, "entry requires one entry operation")
	}
	operation := operations[0]
	if operation.Name == "EntersWithNumberCounters" {
		return compileFixedEntryCounters(operation)
	}
	if operation.Kind != Symbol || operation.Name != "EntersTapped" {
		return nil, unsupported(operation, "entry operation")
	}
	return func() mage.CardOption { return mage.WithKeyword(core.EntersTapped) }, nil
}

func compileFixedEntryCounters(n Node) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	value := n.Args[0]
	if value.Name != "Integer" {
		return nil, unsupported(value, "fixed entry counter amount")
	}
	if err := arity(value, 1); err != nil {
		return nil, err
	}
	amount := value.Args[0]
	if amount.Kind != IntegerNode || amount.Integer < 0 {
		return nil, problem(amount, "entry counter amount requires a nonnegative integer")
	}
	if amount.Integer > math.MaxUint8 {
		return nil, problem(amount, "entry counter amount exceeds engine counter storage")
	}
	counter, err := compilePermanentCounterType(n.Args[1])
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.EntersWithNCounters(counter, amount.Integer))
	}, nil
}
