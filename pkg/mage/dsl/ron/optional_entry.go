package ron

import "github.com/benprew/mage-go/pkg/mage"

type entryAbilityFactory func() mage.AsEntersBattlefieldAbility

func compileOptionalEntry(n Node) (entryAbilityFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	operations, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	for _, operation := range operations {
		if operation.Name == "EnterAsACopyOfAPermanent" {
			return nil, unsupported(operation, "copy inside an ordinary entry sequence")
		}
	}
	if len(operations) == 0 {
		return nil, problem(n, "optional entry requires at least one entry operation")
	}
	factories := make([]entryAbilityFactory, len(operations))
	for i, operation := range operations {
		factories[i], err = compileOptionalEntryOperation(operation)
		if err != nil {
			return nil, err
		}
	}
	return func() mage.AsEntersBattlefieldAbility {
		abilities := make([]mage.AsEntersBattlefieldAbility, len(factories))
		for i, factory := range factories {
			abilities[i] = factory()
		}
		return mage.MayEntryOperations(abilities...)
	}, nil
}

func compileOptionalEntryOperation(n Node) (entryAbilityFactory, error) {
	switch n.Name {
	case "MayActions":
		return compileOptionalEntry(n)
	case "EntersTapped":
		if _, err := symbol(n); err != nil {
			return nil, err
		}
		return mage.EnterTappedAsEnters, nil
	case "ChooseABasicLandType":
		if _, err := symbol(n); err != nil {
			return nil, err
		}
		return mage.ChooseBasicLandTypeAsEnters, nil
	case "ChooseAPlayer":
		if err := validateOpponentEntryOperation(n); err != nil {
			return nil, err
		}
		return mage.ChooseOpponentAsEnters, nil
	default:
		return nil, unsupported(n, "optional entry operation")
	}
}
