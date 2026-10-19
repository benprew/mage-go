package ron

import (
	"maps"
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
)

type selectedPilePlan struct {
	name string
	loop *permanentLoopPlan
}

func compileNamedPileSelection(n Node, remaining []Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody || ctx.namedPiles == nil || ctx.permanentLoop == nil {
		return nil, problem(n, "pile choice requires a preceding partition in a resolving permanent loop")
	}
	if n.Args[1].Kind != Symbol || n.Args[1].Name != "EachablePermanent" {
		return nil, problem(n.Args[1], "pile choice requires the current loop member")
	}
	subject, err := compilePermanentRef(n.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	names, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	if len(names) < 2 {
		return nil, problem(n.Args[0], "pile choice requires at least two distinct declared names")
	}
	indices := make([]int, len(names))
	seen := make(map[string]bool, len(names))
	for i, name := range names {
		if name.Kind != String || seen[name.String] {
			return nil, problem(name, "pile choice requires distinct declared names")
		}
		index := slices.Index(ctx.namedPiles.labels, name.String)
		if index < 0 {
			return nil, problem(name, "unknown pile name")
		}
		indices[i] = index
		seen[name.String] = true
	}
	plan := &selectedPilePlan{name: "selected pile " + n.Position.String(), loop: ctx.permanentLoop}
	ctx.selectedPile = plan
	body, err := compileActionSequence(remaining, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		bindings.selectedPiles = maps.Clone(bindings.selectedPiles)
		if bindings.selectedPiles == nil {
			bindings.selectedPiles = make(map[*selectedPilePlan]*mage.PermanentSetBinding)
		}
		result := mage.BindPermanentSet(plan.name)
		bindings.selectedPiles[plan] = result
		piles := make([]mage.NamedPermanentPile, len(indices))
		for i, index := range indices {
			piles[i] = bindings.namedPiles[ctx.namedPiles][index]
		}
		effects := make([]mage.Effect, 0, 1+len(body))
		effects = append(effects, mage.ChooseNamedPermanentPile(ctx.player.build(bindings), subject.build(bindings), result, piles...))
		for _, build := range body {
			effects = append(effects, build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}

func compileSelectedPileRef(n Node, ctx scope) (permanentSetFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody || ctx.selectedPile == nil || ctx.permanentLoop != ctx.selectedPile.loop || n.Args[0].Kind != Symbol || n.Args[0].Name != "EachablePermanent" {
		return nil, problem(n, "selected pile requires a preceding choice for this loop member")
	}
	return func(bindings instanceBindings) mage.PermanentSet {
		return mage.FromPermanentSet(bindings.selectedPiles[ctx.selectedPile])
	}, nil
}
