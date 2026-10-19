package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileExplicitCardMovement(n Node, ctx scope) (effectFactory, error) {
	if !ctx.resolving || ctx.drawReplacementBody {
		return nil, problem(n, "card movement requires an ordinary resolving body")
	}
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	player, err := compilePlayerRef(n.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	if n.Name == "PutCardFromAnywhereIntoPlayersGraveyard" {
		if n.Args[0].Kind != Symbol || n.Args[0].Name != "ThisCard" || !ctx.permanent {
			return nil, problem(n.Args[0], "from-anywhere movement requires the physical permanent source ThisCard")
		}
		return func(bindings instanceBindings) mage.Effect {
			return mage.PutSourceCardInGraveyardFromAnywhere(player.build(bindings))
		}, nil
	}
	card, err := compileRevealedHandCardRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.PutHandCardInHandFor(card(bindings), player.build(bindings))
	}, nil
}
