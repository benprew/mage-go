package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileSpellTargetRestriction(n, operation Node, ctx scope) (optionFactory, error) {
	return compileConditionalSpellTargetRestriction(n, operation, ctx, nil)
}

func compileConditionalSpellTargetRestriction(n, operation Node, ctx scope, condition mage.StateCondition) (optionFactory, error) {
	if err := arity(operation, 1); err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if subject.kind != permanentSource && subject.kind != permanentHost {
		return nil, unsupported(n.Args[0], "static spell targeting reference")
	}
	filter, err := compileSpellFilter(operation.Args[0])
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		if condition != nil {
			return mage.WithAbility(mage.Static(mage.CantBeTargetedBySpellsIf(subject.build(instanceBindings{}), filter, condition)))
		}
		return mage.WithAbility(mage.Static(mage.CantBeTargetedBySpells(subject.build(instanceBindings{}), filter)))
	}, nil
}
