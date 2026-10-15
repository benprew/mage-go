package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileContextualCollection(n Node, ctx scope) (permanentSetFactory, error) {
	if isLeastPowerCollection(n) {
		return compileLeastPowerCollection(n, ctx)
	}
	switch n.Name {
	case "PutOntoBattlefieldByPermanent":
		return compileEntryHistory(n, ctx)
	case "PermanentPutACounterOfTypeOnIt", "PermanentHasNotRemovedACounterOfTypeFromIt":
		return compileCounterHistory(n, ctx)
	case "TheCreatedTokens":
		if n.Kind != Symbol || ctx.createdTokenLink == nil || !ctx.permanent || !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
			return nil, problem(n, "created tokens require a linked permanent-source resolving collection")
		}
		return func(instanceBindings) mage.PermanentSet {
			return mage.TokensCreatedWith(ctx.createdTokenLink, mage.This)
		}, nil
	case "IsColor":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		value := n.Args[0]
		if value.Name == "TheChosenColor" {
			if value.Kind != Symbol || !ctx.permanent || !ctx.colorEntryChoice || !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
				return nil, problem(value, "chosen color requires a source entry choice in a resolving collection")
			}
			return func(instanceBindings) mage.PermanentSet {
				return mage.PermanentsOfChosenColor(mage.This, mage.MatchingPermanents(mage.AnyPermanent))
			}, nil
		}
		filter, err := compileFilter(n)
		if err != nil {
			return nil, err
		}
		return func(instanceBindings) mage.PermanentSet { return mage.MatchingPermanents(filter) }, nil

	case "SharesACardtypeWithPermanent", "SharesACardtypeWithPermanentFromAmongCardtypes":
		return compilePermanentTypeComparison(n, ctx)
	case "CostWasntPaidForIt":
		if n.Kind != Symbol || ctx.permanentPayment == nil {
			return nil, problem(n, "unbound unpaid permanent set")
		}
		return func(bindings instanceBindings) mage.PermanentSet {
			return mage.FromPermanentSet(bindings.permanentPayments[ctx.permanentPayment])
		}, nil

	case "Trigger_ThosePermanents":
		if n.Kind != Symbol || ctx.statePermanents == nil {
			return nil, problem(n, "unbound state permanent set")
		}
		return func(bindings instanceBindings) mage.PermanentSet {
			return mage.FromPermanentSet(bindings.statePermanents[ctx.statePermanents])
		}, nil

	case "NotChosenThisWay":
		if n.Kind != Symbol {
			return nil, problem(n, "expected plural choice exclusion symbol")
		}
		if ctx.permanentSetChoice == nil {
			return nil, problem(n, "unbound plural permanent choice")
		}
		return func(bindings instanceBindings) mage.PermanentSet {
			return mage.WithoutPermanents(mage.MatchingPermanents(mage.AnyPermanent), mage.FromPermanentSet(bindings.permanentSetChoices[ctx.permanentSetChoice]))
		}, nil
	case "ControlledSinceBeginningOfTurn", "NotControlledSinceBeginningOfTurn":
		filter, err := compileCurrentTurnControlFilter(n)
		if err != nil {
			return nil, err
		}
		return func(instanceBindings) mage.PermanentSet { return mage.MatchingPermanents(filter) }, nil
	case "ExceptFor":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		if n.Args[0].Name != "NotControlledSinceBeginningOfTurn" {
			return nil, unsupported(n.Args[0], "control history exclusion")
		}
		filter, err := compileCurrentTurnControlFilter(n.Args[0])
		if err != nil {
			return nil, err
		}
		return func(instanceBindings) mage.PermanentSet { return mage.MatchingPermanents(mage.Not(filter)) }, nil
	case "ControlledByPlayer":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		if n.Args[0].Kind != Symbol || (n.Args[0].Name != "TheActivePlayer" && n.Args[0].Name != "ActionPlayer") {
			return nil, unsupported(n.Args[0], "collection controller reference")
		}
		player, err := compilePlayerRef(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.PermanentSet {
			return mage.PermanentsControlledBy(player.build(bindings), mage.AnyPermanent)
		}, nil
	case "DealtDamageToPermanentThisTurn":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		ref, err := compilePermanentRef(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.PermanentSet {
			return mage.PermanentsThatDealtDamageToThisTurn(ref.build(bindings), mage.MatchingPermanents(mage.AnyPermanent))
		}, nil
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
	case "WasDealtDamageByPermanentThisTurn":
		ref, err := compileDamageHistorySource(n, ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.PermanentSet {
			if ref.creaturesOnly {
				return mage.PermanentsDealtDamageAsCreaturesByThisTurn(mage.DamageSourceOf(ref.build(bindings)), mage.MatchingPermanents(mage.AnyPermanent))
			}
			return mage.PermanentsDealtDamageByThisTurn(mage.DamageSourceOf(ref.build(bindings)), mage.MatchingPermanents(mage.AnyPermanent))
		}, nil
	case "WasTappedToPayForAbilitiesOfPermanent":
		if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
			return nil, problem(n, "paid tap history requires an ordinary resolving collection")
		}
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		ref, err := compilePermanentRef(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.PermanentSet {
			return mage.TappedForAbilitiesThisTurn(ref.build(bindings))
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
	case "IsBlockingAttacker", "IsBlockedByDefender", "BandedWith":
		if n.Name == "BandedWith" && (!ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody) {
			return nil, problem(n, "band membership requires a resolving collection")
		}
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		ref, err := compilePermanentRef(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.PermanentSet {
			if n.Name == "BandedWith" {
				return mage.BandedWith(ref.build(bindings))
			}
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
