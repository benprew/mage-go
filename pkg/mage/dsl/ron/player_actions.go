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
	if err := arity(n, 2); err != nil {
		return nil, nil, err
	}
	player, err := compilePlayerRef(n.Args[0], ctx)
	if err != nil {
		return nil, nil, err
	}
	parts, err := list(n.Args[1])
	if err != nil {
		return nil, nil, err
	}
	plan := &playerActionsPlan{name: "action player " + n.Position.String()}
	ctx.actionPlayer = plan
	ctx.player = playerReferencePlan{action: plan}
	actions := make([]effectFactory, 0, len(parts))
	for _, part := range parts {
		switch part.Name {
		case "TakeAnExtraTurn", "ChooseACardName", "AnteTopCardOfLibrary", "CreateTokens", "ChooseAnAction", "FlipACoin_OnLose", "FlipACoin_OnWinAndLose", "MillNumberCards", "PlayerActions", "PlayerAction", "TapEachPermanent", "LoseUnspentMana", "DrawACard", "DrawNumberCards", "GainLife", "GainLifeForEach", "LoseLife", "DiscardACard", "DiscardNumberCards", "DiscardACardAtRandom", "DiscardNumberCardsAtRandom", "DiscardHand", "DiscardCards", "RevealHand":
		default:
			return nil, nil, unsupported(part, "scoped player action")
		}
		if part.Name == "ChooseACardName" {
			build, result, choiceErr := compileCardNameChoiceWithResult(part, ctx)
			if choiceErr != nil {
				return nil, nil, choiceErr
			}
			ctx.chosenName = result
			actions = append(actions, build)
			continue
		}
		if part.Name == "PlayerActions" {
			build, result, nestedErr := compilePlayerActionsWithName(part, ctx)
			if nestedErr != nil {
				return nil, nil, nestedErr
			}
			ctx.chosenName = result
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
			return nil, nil, err
		}
		actions = append(actions, body...)
	}
	return func(bindings instanceBindings) mage.Effect {
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
	}, ctx.chosenName, nil
}
