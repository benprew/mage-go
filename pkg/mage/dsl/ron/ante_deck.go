package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileDeckConstruction(n Node) (optionFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	operation := n.Args[0]
	if operation.Kind != Symbol || operation.Name != "RemoveFromDeckIfNotPlayingForAnte" {
		return nil, unsupported(operation, "deck construction operation")
	}
	return mage.WithAnteDeckRestriction, nil
}
