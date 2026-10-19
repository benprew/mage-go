package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileLibrarySearch(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	steps, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	if len(steps) != 3 {
		return nil, problem(n, "single-card search requires find, destination and shuffle")
	}
	if steps[2].Kind != Symbol || steps[2].Name != "Shuffle" {
		return nil, unsupported(steps[2], "search shuffle instruction")
	}
	find, destination := steps[0], steps[1]
	if destination.Name == "IfElse" {
		return compileLibraryManaDifference(n, steps, ctx)
	}
	if find.Name == "FindAGenericCard" {
		if find.Kind != Symbol {
			return nil, unsupported(find, "unqualified search instruction")
		}
		if destination.Kind != Symbol || destination.Name != "PutFoundCardsIntoHand" {
			return nil, unsupported(destination, "unqualified search destination")
		}
		return func(instanceBindings) mage.Effect {
			return (mage.LibrarySearchInstruction{Kind: mage.SearchAnyCardToHand}).Effect()
		}, nil
	}
	if find.Name != "FindACardOfType" {
		return nil, unsupported(find, "single-card search instruction")
	}
	if err := arity(find, 1); err != nil {
		return nil, err
	}
	_, err = compileBasicLandSearchFilter(find.Args[0])
	if err != nil {
		return nil, err
	}
	if destination.Name != "PutFoundCardsOntoBattlefield" {
		return nil, unsupported(destination, "basic land search destination")
	}
	if err := arity(destination, 1); err != nil {
		return nil, err
	}
	entry, err := list(destination.Args[0])
	if err != nil {
		return nil, err
	}
	if len(entry) != 1 || entry[0].Kind != Symbol || entry[0].Name != "EntersNormally" {
		return nil, unsupported(destination.Args[0], "search entry operations")
	}
	return func(instanceBindings) mage.Effect {
		return (mage.LibrarySearchInstruction{Kind: mage.SearchBasicLandToBattlefield}).Effect()
	}, nil
}

func compileBasicLandSearchFilter(n Node) (mage.CardFilter, error) {
	if n.Name != "And" {
		return mage.CardFilter{}, unsupported(n, "basic land search filter")
	}
	if err := arity(n, 1); err != nil {
		return mage.CardFilter{}, err
	}
	parts, err := list(n.Args[0])
	if err != nil {
		return mage.CardFilter{}, err
	}
	if len(parts) != 2 {
		return mage.CardFilter{}, unsupported(n, "basic land search filter")
	}
	basic, land := false, false
	for _, part := range parts {
		if err := arity(part, 1); err != nil {
			return mage.CardFilter{}, err
		}
		value, err := symbol(part.Args[0])
		if err != nil {
			return mage.CardFilter{}, err
		}
		switch {
		case part.Name == "IsSupertype" && value == "Basic" && !basic:
			basic = true
		case part.Name == "IsCardtype" && value == "Land" && !land:
			land = true
		default:
			return mage.CardFilter{}, unsupported(part, "basic land search filter")
		}
	}
	if !basic || !land {
		return mage.CardFilter{}, unsupported(n, "basic land search filter")
	}
	return mage.NewCardFilter("basic land", func(c mage.Card) bool { return c.HasType(core.TypeLand) && c.HasSuperType(core.SuperBasic) }), nil
}
