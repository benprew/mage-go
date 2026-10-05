package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileNumberedPermanentTarget(n Node, ctx scope) (targetFactory, string, error) {
	if err := arity(n, 2); err != nil {
		return nil, "", err
	}
	count := n.Args[0]
	usesX := count.Kind == Symbol && count.Name == "ValueX"
	if usesX && !ctx.x {
		return nil, "", problem(count, "unbound ValueX")
	}
	amount := 0
	if !usesX {
		if count.Name != "Integer" {
			return nil, "", unsupported(count, "announcement target count")
		}
		if err := arity(count, 1); err != nil {
			return nil, "", err
		}
		if count.Args[0].Kind != IntegerNode || count.Args[0].Integer < 0 {
			return nil, "", problem(count, "target count requires a nonnegative integer")
		}
		amount = count.Args[0].Integer
	}
	filter, options, err := compilePermanentTargetFilter(n.Args[1], ctx)
	if err != nil {
		return nil, "", err
	}
	return func() mage.Target {
		name := ctx.targetNamespace + "Ref_TargetPermanents"
		if usesX {
			return mage.DeclareXPermanentTargets(name, filter, options...)
		}
		return mage.DeclarePermanentTargets(name, amount, amount, filter, options...)
	}, "Ref_TargetPermanents", nil
}

func compilePermanentTargetsRef(n Node, ctx scope) (*permanentTargetsPlan, error) {
	if ctx.delayedStep {
		return nil, unsupported(n, "delayed permanent target collection")
	}
	if n.Kind != Symbol || ctx.permanentTargets == nil || n.Name != ctx.permanentTargets.name {
		return nil, problem(n, "unbound or incompatible permanent collection reference")
	}
	return ctx.permanentTargets, nil
}
