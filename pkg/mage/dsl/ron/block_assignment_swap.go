package ron

import "github.com/benprew/mage-go/pkg/mage"

func compilePairedOpponentTargets(n, target Node, ctx scope) (actionPlan, error) {
	var plan actionPlan
	if err := arity(target, 3); err != nil {
		return plan, err
	}
	count := target.Args[0]
	if err := arity(count, 1); err != nil {
		return plan, err
	}
	if count.Name != "Integer" || count.Args[0].Kind != IntegerNode || count.Args[0].Integer != 2 {
		return plan, problem(count, "paired opponent targets require exactly two")
	}
	if target.Args[2].Kind != Symbol || target.Args[2].Name != "ControlledByTheSamePlayer" {
		return plan, unsupported(target.Args[2], "paired target restriction")
	}
	filter, options, err := compilePlayerRelativeTargetFilter(target.Args[1], ctx)
	if err != nil {
		return plan, err
	}
	you, opponent := singularTargetControllers(target.Args[1])
	if you || !opponent {
		return plan, problem(target.Args[1], "paired same-controller targets require opponent control in the two-player engine")
	}
	ctx.permanentTargets = &permanentTargetsPlan{name: "Ref_TargetPermanents", filter: target.Args[1]}
	ctx.pairedTargets = ctx.permanentTargets
	plan, err = compileActions(n.Args[1], ctx)
	if err != nil {
		return plan, err
	}
	if len(plan.targets) != 0 {
		return plan, problem(n, "nested target declarations are not supported")
	}
	plan.targets = []targetFactory{func(bindings instanceBindings) mage.Target {
		return mage.DeclarePermanentTargets(ctx.targetNamespace+ctx.permanentTargets.name, 2, 2, filter, options(bindings)...)
	}}
	plan.permanentTargets = ctx.permanentTargets
	return plan, nil
}

func compileBlockAssignmentSwap(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody || ctx.pairedTargets == nil || ctx.permanentTargets != ctx.pairedTargets {
		return nil, problem(n, "block swap requires a paired opponent target group in an ordinary resolving body")
	}
	first, second := n.Args[0], n.Args[1]
	validPair := (first.Name == "Ref_TargetPermanents_0" && second.Name == "Ref_TargetPermanents_1") || (first.Name == "Ref_TargetPermanents_1" && second.Name == "Ref_TargetPermanents_0")
	if first.Kind != Symbol || second.Kind != Symbol || !validPair {
		return nil, problem(n, "block swap requires the two distinct members of its paired target group")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.TrySwapBlockingAssignments(bindings.permanentSets[ctx.permanentTargets])
	}, nil
}
