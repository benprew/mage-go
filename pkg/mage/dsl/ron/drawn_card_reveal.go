package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

func drawnCardAction(n Node, ctx scope) (Node, playerReferencePlan, error) {
	if n.Name == "PlayerAction" && len(n.Args) == 2 && (n.Args[1].Name == "DrawACard" || n.Args[1].Name == "RevealTopCardOfLibrary") {
		player, err := compilePlayerRef(n.Args[0], ctx)
		return n.Args[1], player, err
	}
	return n, ctx.player, nil
}

func compileDrawnCardProducer(n Node, player playerReferencePlan, remaining []Node, ctx scope) (effectFactory, error) {
	if n.Kind != Symbol {
		return nil, problem(n, n.Name+" requires a symbol")
	}
	drawn := ctx.drawnCard
	if n.Name == "RevealTheCardDrawnThisWay" && drawn == nil {
		return nil, problem(n, "reveal requires a bound drawn card")
	}
	plan := &handRevealPlan{name: n.Name + " " + n.Position.String()}
	if n.Name == "DrawACard" {
		ctx.drawnCard = plan
	} else {
		ctx.handReveal = plan
		ctx.libraryReveal = nil
	}
	body, err := compileActionSequence(remaining, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		bindings.handReveals = maps.Clone(bindings.handReveals)
		if bindings.handReveals == nil {
			bindings.handReveals = make(map[*handRevealPlan]*mage.CardBinding)
		}
		result := mage.BindCard(plan.name)
		bindings.handReveals[plan] = result
		var producer mage.Effect
		if n.Name == "DrawACard" {
			producer = mage.DrawFor(player.build(bindings), mage.Fixed(1), mage.DrawCardInto(result))
		} else {
			producer = mage.RevealHandCardInto(mage.FromCard(bindings.handReveals[drawn]), result)
		}
		effects := make([]mage.Effect, 0, 1+len(body))
		effects = append(effects, producer)
		for _, build := range body {
			effects = append(effects, build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}

func compileDrawnCardDiscard(n Node, ctx scope) ([]effectFactory, error) {
	if n.Kind != Symbol || ctx.drawnCard == nil {
		return nil, problem(n, "discard requires a bound drawn card symbol")
	}
	plan := ctx.drawnCard
	return []effectFactory{func(bindings instanceBindings) mage.Effect {
		return mage.DiscardCardFromHandFor(ctx.player.build(bindings), mage.FromCard(bindings.handReveals[plan]))
	}}, nil
}
