package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileResolvingExileCounterRemoval(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if ctx.sourceZone == nil || ctx.sourceZone.zone != core.ZoneExile {
		return nil, problem(n, "counter removal requires an exile source")
	}
	counter, err := compilePermanentCounterType(n.Args[0])
	if err != nil {
		return nil, err
	}
	card, err := compileCardRef(n.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	if card.sourceZone != ctx.sourceZone {
		return nil, problem(n.Args[1], "counter removal requires the source exile card")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.RemoveCounterFromExiledCard(card.build(bindings), counter)
	}, nil
}
