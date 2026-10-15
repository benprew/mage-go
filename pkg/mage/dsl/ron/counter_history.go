package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileCounterHistory(n Node, ctx scope) (permanentSetFactory, error) {
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "counter history requires an ordinary resolving collection")
	}
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	source, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	counter, err := compilePermanentCounterType(n.Args[1])
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.PermanentSet {
		if n.Name == "PermanentHasNotRemovedACounterOfTypeFromIt" {
			return mage.PermanentsWithoutCountersRemovedBy(source.build(bindings), counter, mage.MatchingPermanents(mage.AnyPermanent))
		}
		return mage.PermanentsGivenCountersBy(source.build(bindings), counter, mage.MatchingPermanents(mage.AnyPermanent))
	}, nil
}

func compileChosenCounterRemoval(n Node, ctx scope) (effectFactory, error) {
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "chosen counter removal requires an ordinary resolving body")
	}
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	counter, err := compilePermanentCounterType(n.Args[0])
	if err != nil {
		return nil, err
	}
	subjects, err := compileContextualCollection(n.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	player := ctx.player
	return func(bindings instanceBindings) mage.Effect {
		choice := mage.BindPermanent("counter removal choice")
		return mage.Sequence(mage.ChoosePermanent(player.build(bindings), subjects(bindings), choice), mage.RemoveAllCounters(mage.From(choice), counter))
	}, nil
}
