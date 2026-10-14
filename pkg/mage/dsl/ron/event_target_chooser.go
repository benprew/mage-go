package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileEventChosenPermanentTarget(n Node, ctx scope, name string) (targetFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	player := n.Args[1]
	if player.Kind != Symbol || player.Name != "Trigger_ThatPlayer" || ctx.stepEvent == nil {
		return nil, problem(player, "target chooser requires a bound step-event player")
	}
	ref, err := compilePlayerRef(player, ctx)
	if err != nil {
		return nil, err
	}
	filter, options, err := compilePlayerRelativeTargetFilter(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Target {
		built := append(options(bindings), mage.TargetChosenBy(ref.build(bindings)))
		return mage.DeclarePermanentTarget(ctx.targetNamespace+name, filter, built...)
	}, nil
}
