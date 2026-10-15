package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileEntryLifeLoss(n Node) (optionFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	value := n.Args[0]
	if err := arity(value, 1); err != nil {
		return nil, err
	}
	var amount func() mage.ResolutionValue
	switch value.Name {
	case "Integer":
		if value.Args[0].Kind != IntegerNode {
			return nil, problem(value.Args[0], "entry life loss requires an integer")
		}
		amount = func() mage.ResolutionValue { return mage.Fixed(value.Args[0].Integer) }
	case "LifeTotalOfPlayer":
		player := value.Args[0]
		if player.Kind != Symbol || player.Name != "You" {
			return nil, unsupported(player, "entry life total player")
		}
		amount = func() mage.ResolutionValue { return mage.LifeTotalOf(mage.You) }
	default:
		return nil, unsupported(value, "entry life loss value")
	}
	return func() mage.CardOption { return mage.WithAbility(mage.LoseLifeAsEnters(amount())) }, nil
}
