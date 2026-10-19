package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileOutsideGameEntry(n Node, ctx scope) (effectFactory, error) {
	if !ctx.resolving || ctx.enteringSource {
		return nil, unsupported(n, "outside-game entry context")
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	filter := n.Args[0]
	if filter.Kind != Symbol || filter.Name != "AnyCard" {
		return nil, unsupported(filter, "outside-game card filter")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.PutOutsideGameCardInHand(ctx.player.build(bindings), mage.CardFilter{})
	}, nil
}
