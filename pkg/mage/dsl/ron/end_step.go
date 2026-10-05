package ron

import "github.com/benprew/mage-go/pkg/mage"

type stepEventPlan struct{ playerName string }

func compileEndStepTrigger(event, body Node, ctx scope) (optionFactory, error) {
	return compileBoundStepTrigger(event, body, ctx, mage.EachEndStep, mage.YourEndStep, "end-step")
}

func compileBoundStepTrigger(event, body Node, ctx scope, each, your mage.TurnStepPattern, label string) (optionFactory, error) {
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	players := event.Args[0]
	var step mage.TurnStepPattern
	if players.Kind == Symbol && players.Name == "AnyPlayer" {
		step = each
	} else {
		if players.Name != "SinglePlayer" {
			return nil, unsupported(players, label+" player filter")
		}
		if err := arity(players, 1); err != nil {
			return nil, err
		}
		player := players.Args[0]
		if player.Kind != Symbol || player.Name != "You" {
			return nil, unsupported(player, label+" player reference")
		}
		step = your
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
