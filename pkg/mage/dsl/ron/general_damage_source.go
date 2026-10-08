package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

func compileGeneralDamageSource(n Node, remaining []Node, ctx scope) (effectFactory, error) {
	plan, err := compileChosenDamageSourceChoice(n)
	if err != nil {
		return nil, err
	}
	ctx.chosenDamageSource = plan
	body, err := compileActionSequence(remaining, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		binding := mage.BindDamageSource("chosen source " + n.Position.String())
		bindings.chosenDamageSources = maps.Clone(bindings.chosenDamageSources)
		if bindings.chosenDamageSources == nil {
			bindings.chosenDamageSources = make(map[*chosenDamageSourcePlan]*mage.DamageSourceBinding)
		}
		bindings.chosenDamageSources[plan] = binding
		effects := make([]mage.Effect, 0, 1+len(body))
		effects = append(effects, mage.ChooseDamageSource(mage.You, plan.filter, binding))
		for _, build := range body {
			effects = append(effects, build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}
