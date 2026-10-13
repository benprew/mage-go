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
	if len(operations) == 0 {
		return nil, problem(n.Args[1], "supply at least one all-player rule operation")
	}
	if len(operations) > 1 {
		factories := make([]optionFactory, 0, len(operations))
		for _, operation := range operations {
			single := n
			single.Args = append([]Node(nil), n.Args...)
			single.Args[1].Args = []Node{operation}
			factory, err := compileAllPlayerRule(single)
			if err != nil {
				return nil, err
			}
			factories = append(factories, factory)
		}
		return func() mage.CardOption {
			options := make([]mage.CardOption, len(factories))
			for i, factory := range factories {
				options[i] = factory()
			}
			return func(card *mage.BaseCard) {
				for _, option := range options {
					option(card)
				}
			}
		}, nil
	}
	operation := operations[0]
	if operation.Name == "CantCastSpells" {
		return compileSpellCastRestriction(n)
	}
	if operation.Name == "CantPlayLands" || operation.Name == "CantPlayLandsOfType" {
		return compileLandPlayRestriction(n)
	}
	if operation.Name == "SkipsUntapStep" {
		if n.Args[0].Kind != Symbol || n.Args[0].Name != "AnyPlayer" {
			return nil, unsupported(n.Args[0], "untap skip player filter")
		}
		if _, err := symbol(operation); err != nil {
			return nil, err
		}
		return func() mage.CardOption { return mage.WithAbility(mage.Static(mage.SkipUntapSteps())) }, nil
	}
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
