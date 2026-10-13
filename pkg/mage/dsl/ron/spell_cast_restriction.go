package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileSpellCastRestriction(n Node) (optionFactory, error) {
	if n.Args[0].Kind != Symbol || n.Args[0].Name != "AnyPlayer" {
		return nil, unsupported(n.Args[0], "spell cast restriction player filter")
	}
	operation := n.Args[1].Args[0]
	if err := arity(operation, 1); err != nil {
		return nil, err
	}
	filter, err := compileOriginalPrintingCardFilter(operation.Args[0], "casting card filter")
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption { return mage.WithAbility(mage.Static(mage.PreventSpellCasts(filter))) }, nil
}
