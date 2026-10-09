package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileAnteExchangeAction(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	card, err := compileCardRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if card.target == nil || !card.target.ante {
		return nil, problem(n.Args[0], "ante action requires an ante card reference")
	}
	player, err := compilePlayerRef(n.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		if n.Name == "AcquireOwnershipOfCard" {
			return mage.AcquireCardOwnership(card.build(bindings), player.build(bindings))
		}
		return mage.ExchangeAnteCardWithLibraryTop(card.build(bindings), player.build(bindings))
	}, nil
}
