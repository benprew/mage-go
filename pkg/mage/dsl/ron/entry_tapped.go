package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileTappedEntry(n Node) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	source := n.Args[0]
	if source.Kind != Symbol || source.Name != "ThisPermanent" {
		return nil, unsupported(source, "tapped entry source")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 {
		return nil, problem(n, "tapped entry requires one entry operation")
	}
	operation := operations[0]
	if operation.Kind != Symbol || operation.Name != "EntersTapped" {
		return nil, unsupported(operation, "entry operation")
	}
	return func() mage.CardOption { return mage.WithKeyword(core.EntersTapped) }, nil
}
