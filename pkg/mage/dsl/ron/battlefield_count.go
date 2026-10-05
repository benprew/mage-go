package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileBattlefieldCount(n Node, ctx scope) (valueFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	var controller *playerReferencePlan
	filter, err := compileCountFilter(n.Args[0], ctx, &controller)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.ResolutionValue {
		if controller != nil {
			return mage.CountPermanents(mage.PermanentsControlledBy(controller.build(bindings), filter))
		}
		return mage.CountPermanents(mage.MatchingPermanents(filter))
	}, nil
}

func compileCountFilter(n Node, ctx scope, controller **playerReferencePlan) (mage.PermanentFilter, error) {
	if n.Name == "ControlledByAPlayer" {
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, err
		}
		player := n.Args[0]
		if player.Name != "SinglePlayer" {
			return mage.PermanentFilter{}, unsupported(player, "count controller")
		}
		if err := arity(player, 1); err != nil {
			return mage.PermanentFilter{}, err
		}
		ref, err := compilePlayerRef(player.Args[0], ctx)
		if err != nil {
			return mage.PermanentFilter{}, err
		}
		if *controller != nil {
			return mage.PermanentFilter{}, problem(n, "multiple count controller restrictions are not supported")
		}
		*controller = &ref
		return mage.AnyPermanent, nil
	}
	if n.Name == "And" {
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return mage.PermanentFilter{}, err
		}
		filters := make([]mage.PermanentFilter, len(parts))
		for i, part := range parts {
			filters[i], err = compileCountFilter(part, ctx, controller)
			if err != nil {
				return mage.PermanentFilter{}, err
			}
		}
		return mage.And(filters...), nil
	}
	return compileFilter(n)
}
