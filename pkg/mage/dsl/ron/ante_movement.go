package ron

import "github.com/benprew/mage-go/pkg/mage"

type anteCardsFactory func(instanceBindings) mage.AnteCardSet

func compileAnteCards(n Node, ctx scope) (anteCardsFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	if n.Name == "Other" {
		if !ctx.anteSourcePayment || ctx.delayedStep {
			return nil, problem(n, "other ante cards require a paid source-ante continuation")
		}
		subjects, err := compileAnteCards(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.AnteCardSet { return mage.OtherAnteCards(subjects(bindings)) }, nil
	}
	if n.Name != "OwnedByAPlayer" {
		return nil, unsupported(n, "ante card filter")
	}
	players := n.Args[0]
	if players.Name != "SinglePlayer" {
		return nil, unsupported(players, "ante owner filter")
	}
	if err := arity(players, 1); err != nil {
		return nil, err
	}
	owner, err := compilePlayerRef(players.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.AnteCardSet { return mage.AnteCardsOwnedBy(owner.build(bindings)) }, nil
}

func compileAnteGraveyard(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	subjects, err := compileAnteCards(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect { return mage.PutAnteCardsInGraveyards(subjects(bindings)) }, nil
}
