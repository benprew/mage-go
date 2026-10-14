package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

type discardCountPlan struct{ name string }

func compileDiscardAmount(n Node, ctx scope) (valueFactory, error) {
	if n.Name == "DiscardACard" {
		if _, err := symbol(n); err != nil {
			return nil, err
		}
		return func(instanceBindings) mage.ResolutionValue { return mage.Fixed(1) }, nil
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	value := n.Args[0]
	if value.Name == "Integer" {
		if err := arity(value, 1); err != nil {
			return nil, err
		}
		if value.Args[0].Kind != IntegerNode || value.Args[0].Integer < 0 {
			return nil, problem(value, "discard amount must be a nonnegative integer")
		}
	} else if value.Name != "ValueX" || value.Kind != Symbol {
		return nil, unsupported(value, "discard amount")
	}
	return compileInstanceValue(value, ctx)
}

func compileDiscardCountProducer(n Node, remaining []Node, ctx scope) (effectFactory, error) {
	amount, err := compileDiscardAmount(n, ctx)
	if err != nil {
		return nil, err
	}
	plan := &discardCountPlan{name: "discard count " + n.Position.String()}
	ctx.discardCount = plan
	body, err := compileActionSequence(remaining, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		value := amount(bindings)
		bindings.discardCounts = maps.Clone(bindings.discardCounts)
		if bindings.discardCounts == nil {
			bindings.discardCounts = make(map[*discardCountPlan]*mage.NumberBinding)
		}
		result := mage.BindNumber(plan.name)
		bindings.discardCounts[plan] = result
		effects := make([]mage.Effect, 0, 1+len(body))
		effects = append(effects, mage.Discard(ctx.player.build(bindings), value, mage.DiscardCountInto(result)))
		for _, build := range body {
			effects = append(effects, build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}

func compileGraveyardHandReturn(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	amount, err := compileInstanceValue(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if n.Args[0].Name != "NumCardsDiscardedThisWay" {
		return nil, unsupported(n.Args[0], "graveyard hand return amount")
	}
	if n.Args[1].Kind != Symbol || n.Args[1].Name != "AnyCard" {
		return nil, unsupported(n.Args[1], "graveyard hand return filter")
	}
	player, err := compilePlayerRef(n.Args[2], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.ReturnChosenGraveyardCardsToHand(player.build(bindings), amount(bindings), mage.NewCardFilter("any card", func(mage.Card) bool { return true }))
	}, nil
}

func hasDiscardCountRead(nodes []Node) bool {
	for _, n := range nodes {
		if n.Name == "NumCardsDiscardedThisWay" || hasDiscardCountRead(n.Args) {
			return true
		}
	}
	return false
}
