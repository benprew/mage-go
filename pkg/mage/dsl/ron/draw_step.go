package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileDrawStepIntervening(n Node, ctx scope) (optionFactory, error) {
	event := n.Args[0]
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	players := event.Args[0]
	own := false
	if players.Kind != Symbol || players.Name != "AnyPlayer" {
		if players.Name != "SinglePlayer" {
			return nil, unsupported(players, "draw-step player filter")
		}
		if err := arity(players, 1); err != nil {
			return nil, err
		}
		if players.Args[0].Kind != Symbol || players.Args[0].Name != "You" {
			return nil, unsupported(players.Args[0], "draw-step player reference")
		}
		own = true
	}
	condition := n.Args[1]
	if condition.Name != "PermanentPassesFilter" {
		return nil, unsupported(condition, "draw-step intervening-if condition")
	}
	if err := arity(condition, 2); err != nil {
		return nil, err
	}
	if condition.Args[0].Kind != Symbol || condition.Args[0].Name != "ThisPermanent" {
		return nil, unsupported(condition.Args[0], "draw-step intervening-if source")
	}
	filter := condition.Args[1]
	if filter.Kind != Symbol || (filter.Name != "IsTapped" && filter.Name != "IsUntapped") {
		return nil, unsupported(filter, "draw-step intervening-if filter")
	}
	ctx.stepEvent = &stepEventPlan{playerName: "Trigger_ThatPlayer"}
	plan, err := compileActions(n.Args[2], ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		player := mage.BindPlayer(ctx.stepEvent.playerName)
		actions := plan.buildWithBindings(instanceBindings{steps: map[*stepEventPlan]*mage.PlayerBinding{ctx.stepEvent: player}})
		trigger := mage.WhenEvent(mage.TurnStepBegins(mage.EachDrawStep, player), mage.Sequence(actions.effects...), mage.WithInterveningIf(mage.SourceTappedStateWithLKI(filter.Name == "IsTapped"))).(*mage.GenericTriggered)
		if own {
			trigger.AndConditionData(mage.EventPlayerIsController{})
		}
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}
