package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileLandwalkBlockingRule(n Node) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 {
		return nil, unsupported(n.Args[1], "global landwalk blocking operations")
	}
	operation := operations[0]
	if operation.Name != "CanBeBlockedAsThoughItDidntHave" {
		return nil, unsupported(operation, "global landwalk blocking operation")
	}
	if err := arity(operation, 1); err != nil {
		return nil, err
	}
	if operation.Args[0].Name != "Landwalk" {
		return nil, unsupported(operation.Args[0], "blocking override ability")
	}
	keyword, err := compileLandwalkKeyword(operation.Args[0])
	if err != nil {
		return nil, err
	}
	if err := validateGlobalLandwalkFilter(n.Args[0], keyword); err != nil {
		return nil, err
	}
	return func() mage.CardOption { return mage.WithStaticAbility(mage.NullifyLandwalkEffect(keyword)) }, nil
}

func validateGlobalLandwalkFilter(n Node, keyword core.Attr) error {
	// The engine rule applies globally, so narrower filters need a separate contract.
	if n.Name != "And" {
		return unsupported(n, "global landwalk blocking filter")
	}
	if err := arity(n, 1); err != nil {
		return err
	}
	parts, err := list(n.Args[0])
	if err != nil {
		return err
	}
	if len(parts) != 2 {
		return unsupported(n, "global landwalk blocking filter")
	}
	creature, ability := false, false
	for _, part := range parts {
		if err := arity(part, 1); err != nil {
			return err
		}
		switch part.Name {
		case "IsCardtype":
			if part.Args[0].Kind != Symbol || part.Args[0].Name != "Creature" || creature {
				return unsupported(part, "global landwalk creature filter")
			}
			creature = true
		case "HasAbility":
			if part.Args[0].Name != "Landwalk" || ability {
				return unsupported(part, "global landwalk ability filter")
			}
			selected, err := compileLandwalkKeyword(part.Args[0])
			if err != nil {
				return err
			}
			if selected != keyword {
				return problem(part, "landwalk filter and blocking override must match")
			}
			ability = true
		default:
			return unsupported(part, "global landwalk blocking filter")
		}
	}
	if !creature || !ability {
		return unsupported(n, "global landwalk blocking filter")
	}
	return nil
}
