package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileFaceDownCast(n Node, ctx scope) (effectFactory, error) {
	return compileFaceDownCastChoice(n, ctx, false)
}

func compileFaceDownCastChoice(n Node, ctx scope, optional bool) (effectFactory, error) {
	return compileFaceDownCastInto(n, ctx, optional, nil)
}

func compileFaceDownCastInto(n Node, ctx scope, optional bool, result *castResultPlan) (effectFactory, error) {
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, unsupported(n, "face-down casting context")
	}
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	card, err := compileChosenHandCardRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	pt := n.Args[1]
	if pt.Kind != Tuple || pt.Name != "PT" || len(pt.Args) != 2 || pt.Args[0].Kind != IntegerNode || pt.Args[1].Kind != IntegerNode || pt.Args[0].Integer != 2 || pt.Args[1].Integer != 2 {
		return nil, unsupported(pt, "face-down characteristics")
	}
	return func(bindings instanceBindings) mage.Effect {
		if result != nil {
			if optional {
				return mage.MayCastHandCardFaceDownWithoutPayingInto(ctx.player.build(bindings), card(bindings), bindings.castResults[result])
			}
			return mage.CastHandCardFaceDownWithoutPayingInto(ctx.player.build(bindings), card(bindings), bindings.castResults[result])
		}
		if optional {
			return mage.MayCastHandCardFaceDownWithoutPaying(ctx.player.build(bindings), card(bindings))
		}
		return mage.CastHandCardFaceDownWithoutPaying(ctx.player.build(bindings), card(bindings))
	}, nil
}
