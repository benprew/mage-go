package ron

import (
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
	if len(operations) == 2 && operations[0].Name == "ChooseAColor" {
		return compileCombinedEntryChoice(n)
	}
	if len(operations) != 1 {
		return nil, problem(n, "entry requires one entry operation")
	}
	operation := operations[0]
	if operation.Name == "LoseLife" {
		return compileEntryLifeLoss(operation)
	}
	if operation.Name == "ChooseANumberBetween" {
		return compileNumericEntry(operation)
	}
	if operation.Name == "EntersWithLayerEffectOfChoice" {
		return compileEntryVariant(operation)
	}
	if operation.Name == "ChooseAColor" {
		if err := validateColorEntryOperation(operation); err != nil {
			return nil, err
		}
		return func() mage.CardOption { return mage.WithAbility(mage.ChooseColorAsEnters()) }, nil
	}
	if operation.Name == "MayActions" {
		if operation.Kind == Tuple && len(operation.Args) == 1 && operation.Args[0].Kind == List && len(operation.Args[0].Args) == 1 && operation.Args[0].Args[0].Name == "EnterAsACopyOfAPermanent" {
			return compileEntryCopy(operation.Args[0].Args[0])
		}
		factory, err := compileOptionalEntry(operation)
		if err != nil {
			return nil, err
		}
		return func() mage.CardOption { return mage.WithAbility(factory()) }, nil
	}
	if operation.Name == "ChooseABasicLandType" {
		if _, err := symbol(operation); err != nil {
			return nil, err
		}
		return func() mage.CardOption { return mage.WithAbility(mage.ChooseBasicLandTypeAsEnters()) }, nil
	}
	if operation.Name == "ChooseAPlayer" {
		if err := validateOpponentEntryChoice(n); err != nil {
			return nil, err
		}
		return func() mage.CardOption { return mage.WithAbility(mage.ChooseOpponentAsEnters()) }, nil
	}
	if operation.Name == "EntersWithACounter" {
		if err := arity(operation, 1); err != nil {
			return nil, err
		}
		counter, err := compilePermanentCounterType(operation.Args[0])
		if err != nil {
			return nil, err
		}
		return func() mage.CardOption {
			return mage.WithAbility(mage.EntersWithNCounters(counter, 1))
		}, nil
	}
	if operation.Name == "EntersWithNumberCounters" {
		return compileEntryCounters(operation)
	}
	if operation.Kind != Symbol || operation.Name != "EntersTapped" {
		return nil, unsupported(operation, "entry operation")
	}
	return func() mage.CardOption { return mage.WithKeyword(core.EntersTapped) }, nil
}

func compileEntryCounters(n Node) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	value := n.Args[0]
	if value.Kind == Symbol && value.Name == "ValueX" {
		counter, err := compilePermanentCounterType(n.Args[1])
		if err != nil {
			return nil, err
		}
		return func() mage.CardOption {
			return mage.WithAbility(mage.EntersWithCounters(counter, mage.AnnouncedXForEnteringPermanent()))
		}, nil
	}
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
	counter, err := compilePermanentCounterType(n.Args[1])
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.EntersWithNCounters(counter, amount.Integer))
	}, nil
}
