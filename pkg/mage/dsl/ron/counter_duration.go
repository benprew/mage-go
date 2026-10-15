package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileTemporaryLayerDuration(n Node, ctx scope, subject permanentReferencePlan) (func(instanceBindings, []mage.ContinuousModification) mage.Effect, error) {
	if n.Name != "ForAsLongAsPermanentHasACounterOfTypeOnIt" {
		return compileTemporaryDuration(n, ctx)
	}
	if ctx.enteringSource || ctx.drawReplacementBody || !ctx.resolving {
		return nil, problem(n, "counter duration requires an ordinary resolving body")
	}
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if n.Args[0].Kind != Symbol || n.Args[0].Name != "EachablePermanent" {
		return nil, problem(n.Args[0], "counter duration requires the layer recipient")
	}
	counter, err := compilePermanentCounterType(n.Args[1])
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings, values []mage.ContinuousModification) mage.Effect {
		return mage.UntilPermanentHasCounter(subject.build(bindings), counter, values...)
	}, nil
}

func compileCounterDurationLandType(n Node, subject permanentReferencePlan) (func(instanceBindings) mage.ContinuousModification, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	name := n.Args[0]
	if name.Kind != Symbol {
		return nil, unsupported(name, "counter-bound basic land type")
	}
	switch name.Name {
	case "Plains", "Island", "Swamp", "Mountain", "Forest":
	default:
		return nil, unsupported(name, "counter-bound basic land type")
	}
	return func(bindings instanceBindings) mage.ContinuousModification {
		return mage.SetSubtypeFamily(subject.build(bindings), mage.SubtypeLand, name.Name)
	}, nil
}
