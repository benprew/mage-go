package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileLifeExchange(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "life exchange requires an ordinary resolving body")
	}
	other, err := compilePlayerRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.ExchangeLifeTotals(ctx.player.build(bindings), other.build(bindings))
	}, nil
}
