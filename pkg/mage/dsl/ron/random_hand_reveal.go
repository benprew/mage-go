package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

type handRevealPlan struct {
	name       string
	referenced bool
}

func compileRevealedHandCardRef(n Node, ctx scope) (func(instanceBindings) mage.CardRef, error) {
	if n.Kind != Symbol || n.Name != "TheCardRevealedFromHandThisWay" || ctx.handReveal == nil {
		return nil, problem(n, "unbound or incompatible revealed hand card reference")
	}
	plan := ctx.handReveal
	return func(bindings instanceBindings) mage.CardRef { return mage.FromCard(bindings.handReveals[plan]) }, nil
}

func isRandomHandReveal(n Node) bool {
	return n.Name == "PlayerAction" && len(n.Args) == 2 && n.Args[1].Name == "RevealACardFromHandAtRandom"
}

func compileRandomHandReveal(n Node, remaining []Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	player, err := compilePlayerRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if n.Args[1].Kind != Symbol {
		return nil, problem(n.Args[1], "RevealACardFromHandAtRandom requires a symbol")
	}
	plan := &handRevealPlan{name: "revealed hand card " + n.Position.String()}
	ctx.handReveal = plan
	ctx.libraryReveal = nil
	body, err := compileActionSequence(remaining, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		bindings.handReveals = maps.Clone(bindings.handReveals)
		if bindings.handReveals == nil {
			bindings.handReveals = make(map[*handRevealPlan]*mage.CardBinding)
		}
		binding := mage.BindCard(plan.name)
		bindings.handReveals[plan] = binding
		effects := make([]mage.Effect, 0, 1+len(body))
		effects = append(effects, mage.RevealRandomHandCardInto(player.build(bindings), binding))
		for _, build := range body {
			effects = append(effects, build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}

func compileRevealedHandDiscard(n Node, ctx scope) ([]effectFactory, error) {
	if n.Kind != Symbol {
		return nil, problem(n, "DiscardTheCardRevealedThisWay requires a symbol")
	}
	card, err := compileRevealedHandCardRef(Node{Kind: Symbol, Name: "TheCardRevealedFromHandThisWay", Position: n.Position}, ctx)
	if err != nil {
		return nil, err
	}
	return []effectFactory{func(bindings instanceBindings) mage.Effect {
		return mage.DiscardCardFromHandFor(ctx.player.build(bindings), card(bindings))
	}}, nil
}
