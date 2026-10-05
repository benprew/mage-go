package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

type playerActionsPlan struct{ name string }

func compilePlayerActions(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	player, err := compilePlayerRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	parts, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	plan := &playerActionsPlan{name: "action player " + n.Position.String()}
	ctx.actionPlayer = plan
	ctx.player = playerReferencePlan{action: plan}
	actions := make([]effectFactory, 0, len(parts))
	for _, part := range parts {
		switch part.Name {
		case "PlayerActions", "PlayerAction", "TapEachPermanent", "LoseUnspentMana", "DrawACard", "DrawNumberCards", "GainLife", "LoseLife", "DiscardACard", "DiscardNumberCards", "DiscardACardAtRandom", "DiscardNumberCardsAtRandom", "DiscardHand", "DiscardCards", "RevealHand":
		default:
			return nil, unsupported(part, "scoped player action")
		}
		var body []effectFactory
		if part.Name == "DiscardHand" || part.Name == "DiscardCards" || part.Name == "RevealHand" {
			body, err = compilePlayerEffect(part, ctx)
		} else {
			body, err = compileEffect(part, ctx)
		}
		if err != nil {
			return nil, err
		}
		actions = append(actions, body...)
	}
	return func(bindings instanceBindings) mage.Effect {
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
	}, nil
}
