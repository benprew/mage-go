package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileLibraryTopSelection(n Node, ctx scope) (effectFactory, error) {
	if !ctx.resolving || ctx.enteringSource {
		return nil, problem(n, "library selection requires a resolving body")
	}
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	var count valueFactory
	if n.Args[0].Name == "Integer" {
		fixed, err := fixedLibraryInspectionCount(n.Args[0])
		if err != nil {
			return nil, err
		}
		count = func(instanceBindings) mage.ResolutionValue { return mage.Fixed(fixed) }
	} else {
		var err error
		count, err = compileInstanceValue(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 2 {
		return nil, problem(n.Args[1], "library selection requires one top choice and random bottom placement")
	}
	for i, name := range []string{"PutAGenericCardOnTopOfLibrary", "PutTheRemainingCardsOnTheBottomOfLibraryInARandomOrder"} {
		if operations[i].Kind != Symbol || operations[i].Name != name {
			return nil, unsupported(operations[i], "library selection operation")
		}
	}
	return func(bindings instanceBindings) mage.Effect {
		player := ctx.player.build(bindings)
		return mage.SelectLibraryTopAndBottomRestRandomly(player, player, count(bindings))
	}, nil
}
