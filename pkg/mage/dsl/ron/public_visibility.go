package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileAllPlayerRule(n Node) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 {
		return nil, problem(n.Args[1], "supply one all-player rule operation")
	}
	operation := operations[0]
	if operation.Name == "IncreaseSpellCost" {
		return compileStaticSpellCostIncrease(n)
	}
	if operation.Name != "PlaysWithHandRevealed" && operation.Name != "PlaysWithTopOfLibraryRevealed" {
		return compilePlayerUntapLimit(n)
	}
	if n.Args[0].Kind != Symbol || n.Args[0].Name != "AnyPlayer" {
		return nil, unsupported(n.Args[0], "public visibility player filter")
	}
	if _, err := symbol(operation); err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		rule := mage.RevealAllLibraryTops()
		if operation.Name == "PlaysWithHandRevealed" {
			rule = mage.RevealAllHands()
		}
		return mage.WithAbility(mage.Static(rule))
	}, nil
}
