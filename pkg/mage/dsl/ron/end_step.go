package ron

import "github.com/benprew/mage-go/pkg/mage"

type stepEventPlan struct{ playerName string }

func compileEndStepTrigger(event, body Node, ctx scope) (optionFactory, error) {
	return compileBoundStepTrigger(event, body, ctx, mage.EachEndStep, mage.YourEndStep, "end-step")
}

func compileBoundStepTrigger(event, body Node, ctx scope, each, your mage.TurnStepPattern, label string) (optionFactory, error) {
	return compileBoundStepTriggerWithCondition(event, body, ctx, each, your, label, nil)
}

func compileBoundStepTriggerWithCondition(event, body Node, ctx scope, each, your mage.TurnStepPattern, label string, condition mage.StateCondition) (optionFactory, error) {
	build, err := compileBoundStepAbility(event, body, ctx, each, your, label, condition)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption { return mage.WithAbility(build()) }, nil
}

func compileBoundStepAbility(event, body Node, ctx scope, each, your mage.TurnStepPattern, label string, condition mage.StateCondition) (func() *mage.GenericTriggered, error) {
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	players := event.Args[0]
	var step mage.TurnStepPattern
	attached := false
	chosen := false
	opponent := false
	if players.Kind == Symbol && players.Name == "AnyPlayer" {
		step = each
	} else if players.Kind == Symbol && players.Name == "Opponent" && label == "upkeep" {
		step = each
		opponent = true
	} else {
		if players.Name != "SinglePlayer" {
			return nil, unsupported(players, label+" player filter")
		}
		if err := arity(players, 1); err != nil {
			return nil, err
		}
		player := players.Args[0]
		if player.Kind == Symbol && player.Name == "HostController" && label == "upkeep" && ctx.aura {
			attached = true
		} else if player.Kind == Symbol && player.Name == "TheChosenPlayer" && label == "upkeep" {
			if ctx.opponentEntryChoice == nil {
				return nil, problem(player, "unbound opponent entry choice")
			}
			chosen = true
		} else if player.Kind != Symbol || player.Name != "You" {
			return nil, unsupported(player, label+" player reference")
		}
		step = your
		if chosen {
			step = each
		}
	}
	ctx.stepEvent = &stepEventPlan{playerName: "Trigger_ThatPlayer"}
	if attached {
		ctx.permanentEvent = &permanentEventPlan{name: "Trigger_ThatPermanent"}
	}
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() *mage.GenericTriggered {
		player := mage.BindPlayer(ctx.stepEvent.playerName)
		bindings := instanceBindings{steps: map[*stepEventPlan]*mage.PlayerBinding{ctx.stepEvent: player}}
		var pattern mage.EventPattern = mage.TurnStepBegins(step, player)
		if chosen {
			pattern = mage.ChosenPlayerUpkeep(player)
		}
		if opponent {
			pattern = mage.OpponentUpkeep(player)
		}
		if attached {
			host := mage.BindPermanent(ctx.permanentEvent.name)
			bindings.tapped = map[*permanentEventPlan]*mage.PermanentBinding{ctx.permanentEvent: host}
			pattern = mage.AttachedControllerUpkeep(host, player)
		}
		actions := plan.buildWithBindings(bindings)
		var options []mage.TriggerOption
		if condition != nil {
			options = append(options, mage.WithInterveningIf(condition))
		}
		trigger := mage.WhenEvent(pattern, mage.Sequence(actions.effects...), options...).(*mage.GenericTriggered)
		if label == "upkeep" && !attached && !opponent && condition == nil {
			trigger.WithLayeredUpkeepTextRevision()
		}
		actions.effects = nil
		return TriggerA(trigger, actions)
	}, nil
}
