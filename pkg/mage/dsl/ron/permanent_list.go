package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type permanentListPlan struct{ name string }

func compilePermanentList(n Node, remaining []Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "permanent lists require an ordinary resolving body")
	}
	subjects, err := compileContextualCollection(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	plan := &permanentListPlan{name: "permanent list " + n.Position.String()}
	ctx.permanentList = plan
	body, err := compileActionSequence(remaining, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		bindings.permanentLists = maps.Clone(bindings.permanentLists)
		if bindings.permanentLists == nil {
			bindings.permanentLists = make(map[*permanentListPlan]*mage.PermanentSetBinding)
		}
		saved := mage.BindPermanentSet(plan.name)
		bindings.permanentLists[plan] = saved
		effects := make([]mage.Effect, 0, 1+len(body))
		effects = append(effects, mage.SnapshotPermanentSet(subjects(bindings), saved))
		for _, build := range body {
			effects = append(effects, build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}

func compilePermanentListRef(n Node, ctx scope) (permanentSetFactory, error) {
	if n.Kind != Symbol || ctx.permanentList == nil || !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "permanent list requires a preceding producer in this resolving scope")
	}
	return func(bindings instanceBindings) mage.PermanentSet {
		return mage.FromPermanentSet(bindings.permanentLists[ctx.permanentList])
	}, nil
}

func compileIndefinitePluralControl(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "plural control requires an ordinary resolving body")
	}
	subjects, err := compileContextualCollection(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.Until(core.Indefinite, mage.ChangeEachController(subjects(bindings), ctx.player.build(bindings)))
	}, nil
}
