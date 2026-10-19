package ron

import (
	"maps"
	"strings"

	"github.com/benprew/mage-go/pkg/mage"
)

func compileNamedPermanentPartition(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody || ctx.attackGroup == nil {
		return nil, problem(n, "named partition requires a resolving attack-group defender action")
	}
	names, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(names) < 2 {
		return nil, problem(n.Args[1], "named partition requires at least two piles")
	}
	labels := make([]string, len(names))
	seen := make(map[string]bool, len(names))
	for i, name := range names {
		if name.Kind != String || strings.TrimSpace(name.String) == "" || seen[name.String] {
			return nil, problem(name, "pile names must be distinct nonempty strings")
		}
		labels[i] = name.String
		seen[name.String] = true
	}
	ctx.eachablePlayer = &ctx.player
	subjects, err := compileContextualCollection(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		piles := bindings.namedPiles[ctx.namedPiles]
		if ctx.namedPiles == nil {
			piles = make([]mage.NamedPermanentPile, len(labels))
			for i, label := range labels {
				piles[i] = mage.NamedPermanentPile{Name: label, Result: mage.BindPermanentSet("named pile " + n.Position.String() + " " + label)}
			}
		}
		return mage.SeparatePermanentsIntoNamedPiles(ctx.player.build(bindings), subjects(bindings), piles...)
	}, nil
}

type namedPilesPlan struct {
	name   string
	labels []string
}

func compileNamedPartitionContinuation(n Node, remaining []Node, ctx scope) (effectFactory, error) {
	plan := &namedPilesPlan{name: "named piles " + n.Position.String()}
	ctx.namedPiles = plan
	producer, err := compileEachPlayerAction(n, ctx)
	if err != nil {
		return nil, err
	}
	for _, label := range n.Args[1].Args[1].Args {
		plan.labels = append(plan.labels, label.String)
	}
	body, err := compileActionSequence(remaining, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		bindings.namedPiles = maps.Clone(bindings.namedPiles)
		if bindings.namedPiles == nil {
			bindings.namedPiles = make(map[*namedPilesPlan][]mage.NamedPermanentPile)
		}
		piles := make([]mage.NamedPermanentPile, len(plan.labels))
		for i, label := range plan.labels {
			piles[i] = mage.NamedPermanentPile{Name: label, Result: mage.BindPermanentSet(plan.name + " " + label)}
		}
		bindings.namedPiles[plan] = piles
		effects := make([]mage.Effect, 0, len(producer)+len(body))
		for _, build := range producer {
			effects = append(effects, build(bindings))
		}
		for _, build := range body {
			effects = append(effects, build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}
