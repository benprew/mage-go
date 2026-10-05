package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileEachPlayerAction(n Node) ([]effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if n.Args[0].Kind != Symbol || n.Args[0].Name != "AnyPlayer" {
		return nil, unsupported(n.Args[0], "player action collection")
	}
	action := n.Args[1]
	switch action.Name {
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
		if amount.Name != "Integer" {
			return nil, unsupported(amount, "fixed all-player draw amount")
		}
		if err := arity(amount, 1); err != nil {
			return nil, err
		}
		if amount.Args[0].Kind != IntegerNode || amount.Args[0].Integer < 0 {
			return nil, problem(amount, "all-player draw amount must be a nonnegative integer")
		}
		count := amount.Args[0].Integer
		return []effectFactory{func(instanceBindings) mage.Effect {
			return mage.DrawEach(mage.EachPlayer, mage.Fixed(count))
		}}, nil
	default:
		return nil, unsupported(action, "all-player action")
	}
}
