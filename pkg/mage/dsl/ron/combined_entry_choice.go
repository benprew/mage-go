package ron

import "github.com/benprew/mage-go/pkg/mage"

func validateCombinedEntryChoice(n Node) error {
	if err := arity(n, 2); err != nil {
		return err
	}
	if n.Args[0].Kind != Symbol || n.Args[0].Name != "ThisPermanent" {
		return unsupported(n.Args[0], "combined entry source")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return err
	}
	if len(operations) != 2 || operations[0].Name != "ChooseAColor" || operations[1].Name != "ChooseAPlayer" {
		return problem(n, "combined entry requires a color choice then an opponent choice")
	}
	if err := validateColorEntryOperation(operations[0]); err != nil {
		return err
	}
	return validateOpponentEntryOperation(operations[1])
}

func compileCombinedEntryChoice(n Node) (optionFactory, error) {
	if err := validateCombinedEntryChoice(n); err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.RequiredEntryOperations(mage.ChooseColorAsEnters(), mage.ChooseOpponentAsEnters()))
	}, nil
}
