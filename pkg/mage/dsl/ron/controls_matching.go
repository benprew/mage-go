package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileControlsMatchingFilter(n Node) (mage.PermanentFilter, error) {
	if n.Name != "ControlsA" {
		return mage.PermanentFilter{}, unsupported(n, "matching permanent player predicate")
	}
	if err := arity(n, 1); err != nil {
		return mage.PermanentFilter{}, err
	}
	return compileResolvingPropertyFilter(n.Args[0])
}

func compileDefenderControlsRule(n, operation Node, ctx scope) (optionFactory, error) {
	subject, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if subject.kind != permanentSource {
		return nil, unsupported(n.Args[0], "defender control restriction source")
	}
	if err := arity(operation, 1); err != nil {
		return nil, err
	}
	condition := operation.Args[0]
	if condition.Name != "PlayerPassesFilter" {
		return nil, unsupported(condition, "defender control condition")
	}
	if err := arity(condition, 2); err != nil {
		return nil, err
	}
	if condition.Args[0].Kind != Symbol || condition.Args[0].Name != "DefendingPlayer" {
		return nil, unsupported(condition.Args[0], "defender control player")
	}
	filter, err := compileControlsMatchingFilter(condition.Args[1])
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.Static(mage.CantAttackIf(mage.This, mage.PlayerControls(mage.CombatDefendingPlayer(), filter))))
	}, nil
}
