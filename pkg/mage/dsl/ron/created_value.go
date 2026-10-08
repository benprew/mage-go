package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

type createdValuePlan struct{ name string }

func compileCreatedValue(n Node, remaining []Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	value, err := compileInstanceValue(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	plan := &createdValuePlan{name: "created value " + n.Position.String()}
	ctx.createdValue = plan
	body, err := compileActionSequence(remaining, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		operand := value(bindings)
		binding := mage.BindNumber(plan.name)
		bindings.createdValues = maps.Clone(bindings.createdValues)
		if bindings.createdValues == nil {
			bindings.createdValues = make(map[*createdValuePlan]*mage.NumberBinding)
		}
		bindings.createdValues[plan] = binding
		effects := make([]mage.Effect, 0, 1+len(body))
		effects = append(effects, mage.SnapshotNumber(operand, binding))
		for _, build := range body {
			effects = append(effects, build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}
