package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileOptionalEffectBlocking(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "blocking choice requires an ordinary resolving body")
	}
	attackers, err := compileContextualCollection(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	blocker, err := compilePermanentRef(n.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.MayHavePermanentBlockAttacker(ctx.player.build(bindings), blocker.build(bindings), attackers(bindings))
	}, nil
}
