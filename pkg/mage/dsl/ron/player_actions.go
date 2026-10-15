package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

type playerActionsPlan struct{ name string }

func compilePlayerActions(n Node, ctx scope) (effectFactory, error) {
	build, _, err := compilePlayerActionsWithName(n, ctx)
	return build, err
}

func compilePlayerActionsWithName(n Node, ctx scope) (effectFactory, *cardNameChoicePlan, error) {
	build, name, _, err := compilePlayerActionsWithResults(n, ctx)
	return build, name, err
}

func compilePlayerActionsWithResults(n Node, ctx scope) (effectFactory, *cardNameChoicePlan, *libraryRevealPlan, error) {
	if err := arity(n, 2); err != nil {
		return nil, nil, nil, err
	}
	player, err := compilePlayerRef(n.Args[0], ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	parts, err := list(n.Args[1])
	if err != nil {
		return nil, nil, nil, err
	}
	plan := &playerActionsPlan{name: "action player " + n.Position.String()}
	ctx.actionPlayer = plan
	ctx.player = playerReferencePlan{action: plan}
	actions := make([]effectFactory, 0, len(parts))
	for _, part := range parts {
		if part.Name == "MayAction" && len(part.Args) == 1 && (part.Args[0].Name == "DrawACard" || part.Args[0].Name == "SearchLibrary") {
			build, optionalErr := compileOptionalTapChoice(part, ctx)
			if optionalErr != nil {
				return nil, nil, nil, optionalErr
			}
			actions = append(actions, build)
			continue
		}
		switch part.Name {
		case "CreateFutureTrigger", "AttachPermanentToAPermanent", "GainControlOfPermanent":
		case "LoseTheGame", "SacrificeAPermanent", "SacrificeAPermanentOfAPlayersChoice", "ChooseTwoColorWords", "ChooseTwoBasicLandTypes", "RevealNumberCardsFromHandAtRandomAndDiscardEachCardOfType", "RevealHandAndDiscardACardOfTypeAtRandom", "PutTopOfLibraryInHand", "PutTopOfLibraryInGraveyard", "RevealTopCardOfLibrary", "TakeAnExtraTurn", "ChooseACardName", "AnteTopCardOfLibrary", "CreateTokens", "ChooseAnAction", "FlipACoin_OnLose", "FlipACoin_OnWinAndLose", "MillNumberCards", "PlayerActions", "PlayerAction", "TapEachPermanent", "LoseUnspentMana", "DrawACard", "DrawNumberCards", "GainLife", "GainLifeForEach", "LoseLife", "DiscardACard", "DiscardNumberCards", "DiscardACardAtRandom", "DiscardNumberCardsAtRandom", "DiscardHand", "DiscardCards", "RevealHand":
		default:
			return nil, nil, nil, unsupported(part, "scoped player action")
		}
		action, revealPlayer, actionErr := drawnCardAction(part, ctx)
		if actionErr != nil {
			return nil, nil, nil, actionErr
		}
		if action.Name == "RevealTopCardOfLibrary" {
			build, result, revealErr := compileLibraryTopReveal(action, revealPlayer)
			if revealErr != nil {
				return nil, nil, nil, revealErr
			}
			ctx.libraryReveal = result
			ctx.handReveal = nil
			actions = append(actions, build)
			continue
		}
		if part.Name == "ChooseTwoColorWords" || part.Name == "ChooseTwoBasicLandTypes" {
			build, result, choiceErr := compileTextWordChoice(part, ctx)
			if choiceErr != nil {
				return nil, nil, nil, choiceErr
			}
			if result.color {
				ctx.colorWords = result
			} else {
				ctx.landWords = result
			}
			actions = append(actions, build)
			continue
		}
		if part.Name == "ChooseACardName" {
			build, result, choiceErr := compileCardNameChoiceWithResult(part, ctx)
			if choiceErr != nil {
				return nil, nil, nil, choiceErr
			}
			ctx.chosenName = result
			actions = append(actions, build)
			continue
		}
		if part.Name == "PlayerActions" {
			build, result, reveal, nestedErr := compilePlayerActionsWithResults(part, ctx)
			if nestedErr != nil {
				return nil, nil, nil, nestedErr
			}
			ctx.chosenName = result
			if reveal != ctx.libraryReveal {
				ctx.handReveal = nil
			}
			ctx.libraryReveal = reveal
			actions = append(actions, build)
			continue
		}
		var body []effectFactory
		if part.Name == "DiscardHand" || part.Name == "DiscardCards" || part.Name == "RevealHand" {
			body, err = compilePlayerEffect(part, ctx)
		} else {
			body, err = compileEffect(part, ctx)
		}
		if err != nil {
			return nil, nil, nil, err
		}
		actions = append(actions, body...)
	}
	return func(bindings instanceBindings) mage.Effect {
		if bindings.libraryReveals == nil {
			bindings.libraryReveals = make(map[*libraryRevealPlan]*mage.CardBinding)
		}
		if bindings.chosenNames == nil {
			bindings.chosenNames = make(map[*cardNameChoicePlan]*mage.CardNameBinding)
		}
		subject := player.build(bindings)
		binding := mage.BindPlayer(plan.name)
		bindings.actionPlayers = maps.Clone(bindings.actionPlayers)
		if bindings.actionPlayers == nil {
			bindings.actionPlayers = make(map[*playerActionsPlan]*mage.PlayerBinding)
		}
		bindings.actionPlayers[plan] = binding
		effects := make([]mage.Effect, 0, 1+len(actions))
		effects = append(effects, mage.SnapshotPlayer(subject, binding))
		for _, action := range actions {
			effects = append(effects, action(bindings))
		}
		return mage.Sequence(effects...)
	}, ctx.chosenName, ctx.libraryReveal, nil
}
