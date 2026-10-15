package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

type exileControllerCountsPlan struct {
	name      string
	selection Node
}

func hasExileControllerCountRead(nodes []Node) bool {
	for _, n := range nodes {
		if n.Name == "NumPermanentsOfTypeExiledThisWay" || hasExileControllerCountRead(n.Args) {
			return true
		}
	}
	return false
}
func compileExileControllerCounts(n Node, remaining []Node, ctx scope) (effectFactory, error) {
	if ctx.enteringSource || ctx.drawReplacementBody {
		return nil, unsupported(n, "exile count producer context")
	}
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	subjects, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	if len(subjects) != 1 || subjects[0].Name != "EachPermanent" {
		return nil, unsupported(n.Args[0], "exile count subjects")
	}
	subject := subjects[0]
	if err := arity(subject, 1); err != nil {
		return nil, err
	}
	modifiers, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(modifiers) != 0 {
		return nil, unsupported(n.Args[1], "exile count modifiers")
	}
	set, err := compileContextualCollection(subject.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	plan := &exileControllerCountsPlan{name: "exiled counts " + n.Position.String(), selection: subject.Args[0]}
	ctx.exileCounts = plan
	body, err := compileActionSequence(remaining, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		bindings.exileCounts = maps.Clone(bindings.exileCounts)
		if bindings.exileCounts == nil {
			bindings.exileCounts = make(map[*exileControllerCountsPlan]*mage.PlayerNumbersBinding)
		}
		result := mage.BindPlayerNumbers(plan.name)
		bindings.exileCounts[plan] = result
		effects := make([]mage.Effect, 0, 1+len(body))
		effects = append(effects, mage.ExileEachWithControllerCounts(set(bindings), result))
		for _, build := range body {
			effects = append(effects, build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}
func compileExileControllerCountDraw(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	if ctx.exileCounts == nil || ctx.enteringSource || ctx.drawReplacementBody || ctx.delayedStep {
		return nil, problem(n, "unbound or incompatible exile count")
	}
	expected := Node{Kind: Tuple, Name: "And", Args: []Node{{Kind: List, Args: []Node{
		{Kind: Tuple, Name: "IsCardtype", Args: []Node{{Kind: Symbol, Name: "Creature"}}},
		{Kind: Tuple, Name: "ControlledByAPlayer", Args: []Node{{Kind: Tuple, Name: "SinglePlayer", Args: []Node{{Kind: Symbol, Name: "ActionPlayer"}}}}},
	}}}}
	if !sameDeclaration(n.Args[0], expected) {
		return nil, unsupported(n.Args[0], "exile count filter")
	}
	if !selectionGuarantees(ctx.exileCounts.selection, expected.Args[0].Args[0]) {
		return nil, problem(n, "exile producer must select creatures")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.DrawEachPlayerNumber(bindings.exileCounts[ctx.exileCounts])
	}, nil
}
