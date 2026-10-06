package ron

import "github.com/benprew/mage-go/pkg/mage"

type drawEventPlan struct{ playerName string }

func compileDrawEventTrigger(event, body Node, ctx scope) (optionFactory, error) {
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	players := event.Args[0]
	kind := players.Name
	switch {
	case players.Kind == Symbol && (kind == "AnyPlayer" || kind == "Opponent"):
	case kind == "SinglePlayer":
		if err := arity(players, 1); err != nil {
			return nil, err
		}
		player := players.Args[0]
		if player.Kind != Symbol || player.Name != "You" {
			return nil, unsupported(player, "draw-event player reference")
		}
	default:
		return nil, unsupported(players, "draw-event player filter")
	}
	ctx.drawEvent = &drawEventPlan{playerName: "Trigger_ThatPlayer"}
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		player := mage.BindPlayer(ctx.drawEvent.playerName)
		actions := plan.buildWithBindings(instanceBindings{drawPlayers: map[*drawEventPlan]*mage.PlayerBinding{ctx.drawEvent: player}})
		var pattern mage.EventPattern = mage.PlayerDrawsCard(player)
		switch kind {
		case "Opponent":
			pattern = mage.OpponentDrawsCard(player)
		case "SinglePlayer":
			pattern = mage.ControllerDrawsCard(player)
		}
		trigger := mage.WhenEvent(pattern, mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}
