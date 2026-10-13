package ron

import (
	"maps"
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
)

type damageOutcomePlan struct{ label string }
type damageOutcomeBindings struct {
	total   *mage.NumberBinding
	results *mage.DamageResultsBinding
}

func hasDamageOutcomeRead(nodes []Node) bool {
	return slices.ContainsFunc(nodes, func(n Node) bool {
		return n.Name == "TheAmountOfDamageDealtThisWay" || n.Name == "PermanentDealtDamageToPlayerThisWay" || hasDamageOutcomeRead(n.Args)
	})
}

func compileDamageOutcomeProducer(n Node, continuation []Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	if ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "damage outcomes require an ordinary resolving body")
	}
	var source damageSourceFactory
	if n.Name == "PermanentDealsDamage" {
		ref, err := compilePermanentRef(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		source = func(bindings instanceBindings) mage.DamageSourceRef { return mage.DamageSourceOf(ref.build(bindings)) }
	} else {
		var err error
		source, err = compileSelfDamageSource(n.Args[0], ctx, false)
		if err != nil {
			return nil, err
		}
	}
	amount, err := compileInstanceValue(n.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	recipient := n.Args[2]
	var single damageRecipientFactory
	var many func(instanceBindings) mage.DamageRecipientSet
	switch recipient.Name {
	case "EachPlayer":
		if err := arity(recipient, 1); err != nil {
			return nil, err
		}
		players, err := compileDamageOutcomePlayers(recipient.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		many = func(bindings instanceBindings) mage.DamageRecipientSet {
			return mage.PlayerDamageRecipients(players(bindings))
		}
	case "EachPermanent":
		set, err := compilePermanentDamageCollection(recipient, ctx)
		if err != nil {
			return nil, err
		}
		many = func(bindings instanceBindings) mage.DamageRecipientSet {
			return mage.PermanentDamageRecipients(set(bindings))
		}
	default:
		single, err = compileDamagePacketRecipient(recipient, ctx)
		if err != nil {
			return nil, err
		}
	}
	plan := &damageOutcomePlan{label: "damage at " + n.Position.String()}
	ctx.damageOutcome = plan
	body, err := compileActionSequence(continuation, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		bindings.damageOutcomes = maps.Clone(bindings.damageOutcomes)
		if bindings.damageOutcomes == nil {
			bindings.damageOutcomes = make(map[*damageOutcomePlan]damageOutcomeBindings)
		}
		result := damageOutcomeBindings{mage.BindNumber(plan.label + " total"), mage.BindDamageResults(plan.label + " outcomes")}
		bindings.damageOutcomes[plan] = result
		var producer mage.Effect
		if many != nil {
			producer = mage.DealDamageEach(source(bindings), many(bindings), amount(bindings), mage.DamageTotalInto(result.total), mage.DamageResultsInto(result.results))
		} else {
			producer = mage.DealDamageFrom(source(bindings), single(bindings), amount(bindings), mage.DamageAmountInto(result.total), mage.DamageOutcomesInto(result.results))
		}
		effects := make([]mage.Effect, 0, 1+len(body))
		effects = append(effects, producer)
		for _, build := range body {
			effects = append(effects, build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}

func compileDamageOutcomePlayers(n Node, ctx scope) (func(instanceBindings) mage.PlayerSet, error) {
	if n.Kind == Symbol && n.Name == "AnyPlayer" {
		return func(instanceBindings) mage.PlayerSet { return mage.EachPlayer }, nil
	}
	if n.Kind == Symbol && n.Name == "Opponent" {
		return func(instanceBindings) mage.PlayerSet { return mage.OpponentsOf(mage.You) }, nil
	}
	if n.Name != "Other" {
		return nil, unsupported(n, "damage outcome player collection")
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	player, err := compilePlayerRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.PlayerSet { return mage.OpponentsOf(player.build(bindings)) }, nil
}

func compileDamageOutcomeBranch(n Node, ctx scope) (effectFactory, error) {
	if ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "damage outcomes require an ordinary resolving body")
	}
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if ctx.damageOutcome == nil {
		return nil, problem(n, "unbound damage outcome condition")
	}
	condition := n.Args[0]
	if err := arity(condition, 2); err != nil {
		return nil, err
	}
	source, err := compilePermanentRef(condition.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	player, err := compilePlayerRef(condition.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	parts, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(n.Args[1], "damage outcome condition requires actions")
	}
	body, err := compileActionSequence(parts, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		effects := make([]mage.Effect, len(body))
		for i, build := range body {
			effects[i] = build(bindings)
		}
		return mage.IfDamageDealtToPlayer(bindings.damageOutcomes[ctx.damageOutcome].results, mage.DamageSourceOf(source.build(bindings)), player.build(bindings), mage.Sequence(effects...))
	}, nil
}
