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
	anteTarget := card.target != nil && card.target.ante
	exileAcquisition := n.Name == "AcquireOwnershipOfCard" && card.exile != nil
	if !anteTarget && !exileAcquisition {
		return nil, problem(n.Args[0], "ownership requires an ante target or singular exile result; library exchange requires an ante target")
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
