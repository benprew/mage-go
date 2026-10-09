package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileContextualCollection(n Node, ctx scope) (permanentSetFactory, error) {
	switch n.Name {
	case "APermanentWithTheHighestManaValue":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		subjects, err := compileContextualCollection(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.PermanentSet {
			return mage.HighestManaValue(subjects(bindings))
		}, nil
	case "WasBlockedByDefenderThisTurn":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		ref, err := compilePermanentRef(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.PermanentSet {
			return mage.BlockedByThisTurn(ref.build(bindings))
		}, nil

	case "SinglePermanent":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		ref, err := compilePermanentRef(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.PermanentSet {
			return mage.SingletonPermanent(ref.build(bindings))
		}, nil
	case "Ref_TargetPermanents":
		ref, err := compilePermanentTargetsRef(n, ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.PermanentSet {
			return bindings.permanentSets[ref]
		}, nil
	case "IsBlockingAttacker", "IsBlockedByDefender":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		ref, err := compilePermanentRef(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.PermanentSet {
			if n.Name == "IsBlockingAttacker" {
				return mage.BlockersOf(ref.build(bindings))
			}
			return mage.BlockedAttackersOf(ref.build(bindings))
		}, nil

	case "And", "Or":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return nil, err
		}
		if len(parts) == 0 {
			return nil, problem(n, "collection requires at least one operand")
		}
		plans := make([]permanentSetFactory, len(parts))
		for i, part := range parts {
			plans[i], err = compileContextualCollection(part, ctx)
			if err != nil {
				return nil, err
			}
		}
		return func(bindings instanceBindings) mage.PermanentSet {
			sets := make([]mage.PermanentSet, len(plans))
			for i, build := range plans {
				sets[i] = build(bindings)
			}
			if n.Name == "And" {
				return mage.IntersectPermanents(sets...)
			}
			return mage.UnionPermanents(sets...)
		}, nil
	case "AttachedToPermanent":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		host, err := compilePermanentRef(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.PermanentSet {
			return mage.AttachmentsOn(mage.SingletonPermanent(host.build(bindings)))
		}, nil
	case "AttachedToAPermanent":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		hosts, err := compileContextualCollection(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.PermanentSet { return mage.AttachmentsOn(hosts(bindings)) }, nil
	case "OwnedByAPlayer", "ControlledByAPlayer":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		player := n.Args[0]
		if n.Name == "ControlledByAPlayer" && player.Kind == Symbol && player.Name == "Opponent" {
			return func(instanceBindings) mage.PermanentSet {
				return mage.PermanentsControlledByPlayers(mage.OpponentsOf(mage.You), mage.AnyPermanent)
			}, nil
		}
		if player.Name != "SinglePlayer" {
			return nil, unsupported(player, "collection player")
		}
		if err := arity(player, 1); err != nil {
			return nil, err
		}
		ref, err := compilePlayerRef(player.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.PermanentSet {
			if n.Name == "OwnedByAPlayer" {
				return mage.PermanentsOwnedBy(ref.build(bindings), mage.AnyPermanent)
			}
			return mage.PermanentsControlledBy(ref.build(bindings), mage.AnyPermanent)
		}, nil
	default:
		filter, err := compileFilter(n)
		if err != nil {
			return nil, err
		}
		return func(instanceBindings) mage.PermanentSet { return mage.MatchingPermanents(filter) }, nil
	}
}
