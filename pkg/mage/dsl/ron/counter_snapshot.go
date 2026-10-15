package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

type counterSnapshotPlan struct{ label string }

func compileCounterSnapshot(n Node, remaining []Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "counter snapshots require an ordinary resolving body")
	}
	subject, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	plan := &counterSnapshotPlan{label: "counter snapshot " + n.Position.String()}
	ctx.counterSnapshot = plan
	body, err := compileActionSequence(remaining, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		bindings.counterSnapshots = maps.Clone(bindings.counterSnapshots)
		if bindings.counterSnapshots == nil {
			bindings.counterSnapshots = make(map[*counterSnapshotPlan]*mage.CounterSnapshotBinding)
		}
		saved := mage.BindCounterSnapshot(plan.label)
		bindings.counterSnapshots[plan] = saved
		effects := make([]mage.Effect, 0, 1+len(body))
		effects = append(effects, mage.SnapshotCounters(subject.build(bindings), saved))
		for _, build := range body {
			effects = append(effects, build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}
