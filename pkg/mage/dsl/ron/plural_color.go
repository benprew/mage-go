package ron

import (
	"math"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type permanentTargetsPlan struct{ name string }

func compilePluralPermanentTarget(n Node, ctx scope) (targetFactory, string, error) {
	filter, options, err := compilePermanentTargetFilter(n.Args[0], ctx)
	if err != nil {
		return nil, "", err
	}
	return func(instanceBindings) mage.Target {
		return mage.DeclarePermanentTargets(ctx.targetNamespace+"Ref_TargetPermanents", 1, math.MaxInt, filter, options...)
	}, "Ref_TargetPermanents", nil
}

func compilePluralColor(n Node, ctx scope) (effectFactory, error) {
	if ctx.delayedStep {
		return nil, unsupported(n, "delayed plural color modification")
	}
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	ref := n.Args[0]
	if ref.Kind != Symbol || ctx.permanentTargets == nil || ref.Name != ctx.permanentTargets.name {
		return nil, problem(ref, "unbound or incompatible permanent collection reference")
	}
	if n.Args[2].Kind != Symbol || n.Args[2].Name != "UntilEndOfTurn" {
		return nil, unsupported(n.Args[2], "plural color duration")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 {
		return nil, problem(n.Args[1], "supply one plural color modification")
	}
	op := operations[0]
	if op.Name != "SetColor" {
		return nil, unsupported(op, "plural color modification")
	}
	if err := arity(op, 1); err != nil {
		return nil, err
	}
	colorsNode := op.Args[0]
	if colorsNode.Name != "SimpleColorList" {
		return nil, unsupported(colorsNode, "plural color list")
	}
	if err := arity(colorsNode, 1); err != nil {
		return nil, err
	}
	nodes, err := list(colorsNode.Args[0])
	if err != nil {
		return nil, err
	}
	colors := make([]core.Color, len(nodes))
	for i, node := range nodes {
		colors[i], err = color(node)
		if err != nil {
			return nil, err
		}
		if colors[i] == core.Colorless {
			return nil, problem(node, "use an empty color list for colorless")
		}
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.Until(core.EndOfTurn, mage.SetEachColors(bindings.permanentSets[ctx.permanentTargets], colors...))
	}, nil
}
