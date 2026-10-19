package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileSelectedCollectionPrevention(n Node, ctx scope) (effectFactory, error) {
	collection := n.Args[0].Args[0]
	subjects, err := compileContextualCollection(collection, ctx)
	if err != nil {
		return nil, err
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 || operations[0].Kind != Symbol || operations[0].Name != "PreventThatDamage" {
		return nil, problem(n.Args[1], "collection prevention requires the symbol PreventThatDamage")
	}
	if n.Args[2].Kind != Symbol || n.Args[2].Name != "UntilEndOfTurn" {
		return nil, unsupported(n.Args[2], "selected collection prevention duration")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.PreventCombatDamageByEachThisTurn(subjects(bindings))
	}, nil
}
