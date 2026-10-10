package ron

import "github.com/benprew/mage-go/pkg/mage"

func compilePlayerAttackRestriction(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	player, err := compilePlayerRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	duration := n.Args[2]
	if duration.Name != "UntilPlayersNextTurn" {
		return nil, unsupported(duration, "player attack restriction duration")
	}
	if err := arity(duration, 1); err != nil {
		return nil, err
	}
	until, err := compilePlayerRef(duration.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) == 0 {
		return nil, problem(n.Args[1], "player effect requires at least one operation")
	}
	filters := make([]mage.PermanentFilter, len(operations))
	for i, operation := range operations {
		if operation.Name != "CantBeAttackedExceptBy" {
			return nil, unsupported(operation, "temporary player attack operation")
		}
		if err := arity(operation, 1); err != nil {
			return nil, err
		}
		filters[i], err = compileFilter(operation.Args[0])
		if err != nil {
			return nil, err
		}
	}
	return func(bindings instanceBindings) mage.Effect {
		effects := make([]mage.Effect, len(filters))
		for i, filter := range filters {
			effects[i] = mage.CantBeAttackedExceptByUntilNextTurn(player.build(bindings), filter, until.build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}
