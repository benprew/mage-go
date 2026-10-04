package ron

import "github.com/benprew/mage-go/pkg/mage"

type stepEventPlan struct{ playerName string }

func compileEndStepTrigger(event, body Node, ctx scope) (optionFactory, error) {
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	players := event.Args[0]
	var step mage.TurnStepPattern
	if players.Kind == Symbol && players.Name == "AnyPlayer" {
		step = mage.EachEndStep
	} else {
		if players.Name != "SinglePlayer" {
			return nil, unsupported(players, "end-step player filter")
		}
		if err := arity(players, 1); err != nil {
			return nil, err
		}
		player := players.Args[0]
		if player.Kind != Symbol || player.Name != "You" {
			return nil, unsupported(player, "end-step player reference")
		}
		step = mage.YourEndStep
	}
	ctx.stepEvent = &stepEventPlan{playerName: "Trigger_ThatPlayer"}
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		player := mage.BindPlayer(ctx.stepEvent.playerName)
		bindings := instanceBindings{steps: map[*stepEventPlan]*mage.PlayerBinding{ctx.stepEvent: player}}
		actions := plan.buildWithBindings(bindings)
		trigger := mage.WhenEvent(mage.TurnStepBegins(step, player), mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}
