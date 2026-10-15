package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileEachPlayerAction(n Node, ctx scope) ([]effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if n.Args[0].Kind != Symbol || n.Args[0].Name != "AnyPlayer" {
		return nil, unsupported(n.Args[0], "player action collection")
	}
	action := n.Args[1]
	switch action.Name {
	case "DiscardAllButNumberCards":
		build, err := compileDiscardAllBut(action)
		if err != nil {
			return nil, err
		}
		return []effectFactory{build}, nil
	case "SacrificeAllPermanentsExceptForNum":
		build, err := compileSacrificeAllBut(action, ctx)
		if err != nil {
			return nil, err
		}
		return []effectFactory{build}, nil
	case "ShuffleHandAndGraveyardIntoLibrary":
		if action.Kind != Symbol {
			return nil, problem(action, "ShuffleHandAndGraveyardIntoLibrary requires a symbol")
		}
		return []effectFactory{func(instanceBindings) mage.Effect {
			return mage.ShuffleHandsAndGraveyardsIntoLibraries(mage.EachPlayer)
		}}, nil
	case "AnteTopCardOfLibrary":
		if action.Kind != Symbol {
			return nil, problem(action, "AnteTopCardOfLibrary requires a symbol")
		}
		return []effectFactory{func(instanceBindings) mage.Effect {
			return mage.AnteTopEach(mage.EachPlayer)
		}}, nil
	case "DiscardHand":
		if action.Kind != Symbol {
			return nil, problem(action, "DiscardHand requires a symbol")
		}
		return []effectFactory{func(instanceBindings) mage.Effect {
			return mage.DiscardEach(mage.EachPlayer)
		}}, nil
	case "DrawNumberCards":
		if err := arity(action, 1); err != nil {
			return nil, err
		}
		amount := action.Args[0]
		if amount.Name == "NumPermanentsOfTypeExiledThisWay" {
			build, err := compileExileControllerCountDraw(amount, ctx)
			if err != nil {
				return nil, err
			}
			return []effectFactory{build}, nil
		}
		if amount.Name == "Integer" && amount.Kind == Tuple && len(amount.Args) == 1 && amount.Args[0].Kind == IntegerNode && amount.Args[0].Integer < 0 {
			return nil, problem(amount, "all-player draw amount must be a nonnegative integer")
		}
		value, err := compileInstanceValue(amount, ctx)
		if err != nil {
			return nil, err
		}
		return []effectFactory{func(bindings instanceBindings) mage.Effect {
			return mage.DrawEach(mage.EachPlayer, value(bindings))
		}}, nil
	default:
		return nil, unsupported(action, "all-player action")
	}
}

func compileHandShuffleDraw(shuffle, draw Node) (effectFactory, error) {
	if err := arity(shuffle, 2); err != nil {
		return nil, err
	}
	if shuffle.Args[0].Kind != Symbol || shuffle.Args[0].Name != "AnyPlayer" {
		return nil, unsupported(shuffle.Args[0], "shuffle player collection")
	}
	if shuffle.Args[1].Kind != Symbol {
		return nil, problem(shuffle.Args[1], "ShuffleHandIntoLibrary requires a symbol")
	}
	if draw.Name != "EachPlayerAction" {
		return nil, problem(draw, "hand shuffle requires an adjacent all-player draw of its result")
	}
	if err := arity(draw, 2); err != nil {
		return nil, err
	}
	if draw.Args[0].Kind != Symbol || draw.Args[0].Name != "AnyPlayer" {
		return nil, unsupported(draw.Args[0], "shuffle-result player collection")
	}
	action := draw.Args[1]
	if action.Name != "DrawNumberCards" {
		return nil, unsupported(action, "shuffle-result continuation")
	}
	if err := arity(action, 1); err != nil {
		return nil, err
	}
	result := action.Args[0]
	if result.Kind != Symbol || result.Name != "NumCardsShuffledIntoLibraryThisWay" {
		return nil, unsupported(result, "per-player shuffle result")
	}
	return func(instanceBindings) mage.Effect {
		return mage.ShuffleHandsIntoLibrariesAndDraw(mage.EachPlayer)
	}, nil
}
