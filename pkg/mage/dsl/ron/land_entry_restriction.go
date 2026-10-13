package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileLandEntryRestriction(n Node) (optionFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	filter := n.Args[0]
	if filter.Name != "IsCardtype" {
		return nil, unsupported(filter, "entry prohibition card filter")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	if filter.Args[0].Kind != Symbol || filter.Args[0].Name != "Land" {
		return nil, unsupported(filter.Args[0], "entry prohibition card type")
	}
	return func() mage.CardOption { return mage.WithAbility(mage.Static(mage.PreventLandEntry())) }, nil
}
