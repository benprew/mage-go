package ron

import "github.com/benprew/mage-go/pkg/mage"

func isHandSizeRule(n Node) bool {
	if len(n.Args) != 2 || n.Args[1].Kind != List || len(n.Args[1].Args) != 1 {
		return false
	}
	name := n.Args[1].Args[0].Name
	return name == "HasNoMaximumHandSize" || name == "SetMaximumHandSize"
}

func compileHandSizeRule(n Node, ctx scope) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	player, err := symbol(n.Args[0])
	if err != nil {
		return nil, err
	}
	var subject mage.PlayerRef
	switch player {
	case "You":
		subject = mage.You
	case "TheChosenPlayer":
		if ctx.opponentEntryChoice == nil {
			return nil, problem(n.Args[0], "entry-chosen player requires an opponent entry declaration")
		}
		subject = mage.ChosenPlayerOf(mage.This)
	default:
		return nil, unsupported(n.Args[0], "static hand size player")
	}
	operation := n.Args[1].Args[0]
	if operation.Name == "HasNoMaximumHandSize" {
		if _, err := symbol(operation); err != nil {
			return nil, err
		}
		return func() mage.CardOption { return mage.WithStaticAbility(mage.NoMaximumHandSizeStatic(subject)) }, nil
	}
	if err := arity(operation, 1); err != nil {
		return nil, err
	}
	value := operation.Args[0]
	if value.Name != "Integer" {
		return nil, unsupported(value, "fixed maximum hand size")
	}
	if err := arity(value, 1); err != nil {
		return nil, err
	}
	if value.Args[0].Kind != IntegerNode || value.Args[0].Integer < 0 {
		return nil, problem(value, "maximum hand size requires a nonnegative integer")
	}
	size := value.Args[0].Integer
	return func() mage.CardOption { return mage.WithStaticAbility(mage.SetMaximumHandSizeStatic(subject, size)) }, nil
}
