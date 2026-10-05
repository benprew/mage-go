package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileLibraryInspection(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	player, err := compilePlayerRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	count, err := fixedLibraryInspectionCount(n.Args[1])
	if err != nil {
		return nil, err
	}
	operations, err := list(n.Args[2])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 {
		return nil, problem(n.Args[2], "library inspection requires exactly one ordering operation")
	}
	operation := operations[0]
	if operation.Kind != Symbol {
		return nil, unsupported(operation, "library inspection ordering operation")
	}
	reorder := false
	switch operation.Name {
	case "PutTheRemainingCardsOnTopOfLibraryInAnyOrder":
		reorder = true
	case "LeaveRemainingCardsOnTopOfLibraryInSameOrder":
	default:
		return nil, unsupported(operation, "library inspection ordering operation")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.LookAtLibraryTop(mage.You, player.build(bindings), count, reorder)
	}, nil
}

func fixedLibraryInspectionCount(n Node) (int, error) {
	if n.Name != "Integer" {
		return 0, unsupported(n, "fixed library inspection count")
	}
	if err := arity(n, 1); err != nil {
		return 0, err
	}
	if n.Args[0].Kind != IntegerNode || n.Args[0].Integer < 0 {
		return 0, problem(n, "library inspection requires a nonnegative integer")
	}
	return n.Args[0].Integer, nil
}

func compileMayShuffleLibrary(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	player, err := compilePlayerRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	action := n.Args[1]
	if action.Kind != Symbol || action.Name != "Shuffle" {
		return nil, unsupported(action, "optional library action")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.MayShuffleLibrary(mage.You, player.build(bindings))
	}, nil
}
