package ron

import "github.com/benprew/mage-go/pkg/mage"

func isLeastPowerCollection(n Node) bool {
	return n.Name == "PowerIs" && len(n.Args) == 1 && len(n.Args[0].Args) == 1 && n.Args[0].Args[0].Name == "TheLeastPowerAmongPermanents"
}

func compileLeastPowerCollection(n Node, ctx scope) (permanentSetFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	comparison := n.Args[0]
	if comparison.Name != "EqualTo" {
		return nil, unsupported(comparison, "least-power comparison")
	}
	if err := arity(comparison, 1); err != nil {
		return nil, err
	}
	least := comparison.Args[0]
	if err := arity(least, 1); err != nil {
		return nil, err
	}
	compared, err := compileContextualCollection(least.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.PermanentSet {
		return mage.PowerEqualToLeastOf(mage.MatchingPermanents(mage.AnyPermanent), compared(bindings))
	}, nil
}

func compileChosenDestruction(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	subjects, err := compileContextualCollection(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		choice := mage.BindPermanent("permanent to destroy " + n.Position.String())
		return mage.Sequence(mage.ChoosePermanent(ctx.player.build(bindings), subjects(bindings), choice), mage.DestroyWithoutRegeneration(mage.From(choice)))
	}, nil
}
