package ron

import "github.com/benprew/mage-go/pkg/mage"

func isUnlimitedLandPlayRule(n Node) bool {
	if len(n.Args) != 2 || n.Args[1].Kind != List {
		return false
	}
	for _, operation := range n.Args[1].Args {
		if operation.Name == "MayPlayAnyNumberOfLandsDuringThierTurn" {
			return true
		}
	}
	return false
}

func compileUnlimitedLandPlayRule(n Node) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	player, err := symbol(n.Args[0])
	if err != nil {
		return nil, err
	}
	if player != "You" {
		return nil, unsupported(n.Args[0], "unlimited land play player")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 {
		return nil, problem(n.Args[1], "supply one unlimited land play operation")
	}
	if _, err := symbol(operations[0]); err != nil {
		return nil, err
	}
	return func() mage.CardOption { return mage.WithStaticAbility(mage.AllowUnlimitedLandPlaysFor(mage.You)) }, nil
}
