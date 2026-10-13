package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileLandPlayRestriction(n Node) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if n.Args[0].Kind != Symbol || n.Args[0].Name != "AnyPlayer" {
		return nil, unsupported(n.Args[0], "land play restriction player filter")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 {
		return nil, problem(n.Args[1], "supply one land play restriction")
	}
	operation := operations[0]
	filter := mage.NewCardFilter("all land cards", func(mage.Card) bool { return true })
	if operation.Name == "CantPlayLands" {
		if _, err := symbol(operation); err != nil {
			return nil, err
		}
	} else {
		if operation.Name != "CantPlayLandsOfType" {
			return nil, unsupported(operation, "land play restriction")
		}
		if err := arity(operation, 1); err != nil {
			return nil, err
		}
		filter, err = compileLandPlayCardFilter(operation.Args[0])
		if err != nil {
			return nil, err
		}
	}
	return func() mage.CardOption { return mage.WithAbility(mage.Static(mage.PreventLandPlays(filter))) }, nil
}

func compileLandPlayCardFilter(n Node) (mage.CardFilter, error) {
	if n.Kind != Symbol {
		return mage.CardFilter{}, unsupported(n, "land play card filter")
	}
	set := "ARN"
	switch n.Name {
	case "SharesANameOriginallyPrintedInArabianNights":
	case "SharesANameOriginallyPrintedInAntiquities":
		set = "ATQ"
	default:
		return mage.CardFilter{}, unsupported(n, "land play card filter")
	}
	catalog, err := mage.OriginalPrintingNameCatalog(set)
	if err != nil {
		return mage.CardFilter{}, problem(n, err.Error())
	}
	return mage.NewCardFilter("with a name in the supplied catalog", func(card mage.Card) bool {
		return catalog.Matches([]string{card.Name()})
	}), nil
}
