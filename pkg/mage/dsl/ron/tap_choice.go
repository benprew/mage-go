package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileOptionalTapChoice(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	action := n.Args[0]
	if action.Name == "CastCardInHandWithoutPayingAsAFacedownCreatureSpell" {
		return compileFaceDownCastChoice(action, ctx, true)
	}
	if action.Name == "ChooseAnAttackingCreatureForBlockerToBlock" {
		return compileOptionalEffectBlocking(action, ctx)
	}
	if action.Name == "CreatePermanentLayerEffect" {
		return compileOptionalResolvingCopy(action, ctx)
	}
	if action.Name == "SearchLibrary" {
		return compileOptionalBoundedLibrarySearch(action, ctx)
	}
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
	return compileTapAction(action, ctx)
}
