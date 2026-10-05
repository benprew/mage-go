package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileReturnCollection(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	var owner *playerReferencePlan
	filter, err := compileOwnedCollectionFilter(n.Args[0], ctx, &owner)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		if owner != nil {
			return mage.ReturnEachToHand(mage.PermanentsOwnedBy(owner.build(bindings), filter))
		}
		return mage.ReturnEachToHand(mage.MatchingPermanents(filter))
	}, nil
}

func compileOwnedCollectionFilter(n Node, ctx scope, owner **playerReferencePlan) (mage.PermanentFilter, error) {
	switch n.Name {
	case "OwnedByAPlayer":
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, err
		}
		player := n.Args[0]
		if player.Name != "SinglePlayer" {
			return mage.PermanentFilter{}, unsupported(player, "collection owner")
		}
		if err := arity(player, 1); err != nil {
			return mage.PermanentFilter{}, err
		}
		ref, err := compilePlayerRef(player.Args[0], ctx)
		if err != nil {
			return mage.PermanentFilter{}, err
		}
		if *owner != nil {
			return mage.PermanentFilter{}, problem(n, "multiple collection owner restrictions are not supported")
		}
		*owner = &ref
		return mage.AnyPermanent, nil
	case "And":
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return mage.PermanentFilter{}, err
		}
		if len(parts) == 0 {
			return mage.PermanentFilter{}, problem(n, "collection conjunction requires at least one filter")
		}
		filters := make([]mage.PermanentFilter, len(parts))
		for i, part := range parts {
			filters[i], err = compileOwnedCollectionFilter(part, ctx, owner)
			if err != nil {
				return mage.PermanentFilter{}, err
			}
		}
		return mage.And(filters...), nil
	default:
		return compileFilter(n)
	}
}
