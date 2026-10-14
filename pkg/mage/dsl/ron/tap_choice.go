package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileOptionalTapChoice(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	action := n.Args[0]
	if action.Name == "ChooseANumberBetween" {
		return compileOptionalNumericChoice(action, ctx)
	}
	if action.Name == "PutGraveyardCardOntoBattlefield" {
		return compileOptionalSourceReturn(action, ctx)
	}
	if action.Name == "DrawACard" {
		if action.Kind != Symbol {
			return nil, problem(action, "DrawACard requires a symbol")
		}
		return func(bindings instanceBindings) mage.Effect {
			return mage.MayDrawCard(ctx.player.build(bindings))
		}, nil
	}
	if action.Name == "PutACardFromHandOnBattlefield" {
		return compileOptionalHandEntry(action)
	}
	if action.Name != "TapOrUntapPermanent" {
		return nil, unsupported(action, "optional action")
	}
	if err := arity(action, 1); err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(action.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.MayTapOrUntap(subject.build(bindings))
	}, nil
}
