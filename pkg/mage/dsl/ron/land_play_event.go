package ron

import "github.com/benprew/mage-go/pkg/mage"

type landPlayEventPlan struct{ land *permanentEventPlan }

func compileLandPlayIntervening(n Node, ctx scope) (optionFactory, error) {
	event, condition := n.Args[0], n.Args[1]
	if err := arity(event, 2); err != nil {
		return nil, err
	}
	players, filter := event.Args[0], event.Args[1]
	if players.Name != "SinglePlayer" {
		return nil, unsupported(players, "land-play player filter")
	}
	if err := arity(players, 1); err != nil {
		return nil, err
	}
	if players.Args[0].Kind != Symbol || players.Args[0].Name != "You" {
		return nil, unsupported(players.Args[0], "land-play player")
	}
	if filter.Name != "IsCardtype" {
		return nil, unsupported(filter, "land-play filter")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	if filter.Args[0].Kind != Symbol || filter.Args[0].Name != "Land" {
		return nil, unsupported(filter.Args[0], "land-play type")
	}
	if condition.Name != "PermanentPassesFilter" {
		return nil, unsupported(condition, "land-play historical condition")
	}
	if err := arity(condition, 2); err != nil {
		return nil, err
	}
	if condition.Args[0].Kind != Symbol || condition.Args[0].Name != "Trigger_ThatLand" {
		return nil, unsupported(condition.Args[0], "land-play historical subject")
	}
	predicate := condition.Args[1]
	if predicate.Name != "Not" {
		return nil, unsupported(predicate, "land-play historical predicate")
	}
	if err := arity(predicate, 1); err != nil {
		return nil, err
	}
	if predicate.Args[0].Kind != Symbol || predicate.Args[0].Name != "IsFirstLandPlayedByPlayerThisTurn" {
		return nil, unsupported(predicate.Args[0], "land-play ordinal predicate")
	}
	ctx.landPlayEvent = &landPlayEventPlan{land: &permanentEventPlan{name: "Trigger_ThatLand"}}
	plan, err := compileActions(n.Args[2], ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		player, land, ordinal := mage.BindPlayer("land player"), mage.BindPermanent("played land"), mage.BindNumber("land ordinal")
		bindings := instanceBindings{
			landPlayPlayers: map[*landPlayEventPlan]*mage.PlayerBinding{ctx.landPlayEvent: player},
			tapped:          map[*permanentEventPlan]*mage.PermanentBinding{ctx.landPlayEvent.land: land},
		}
		actions := plan.buildWithBindings(bindings)
		trigger := mage.WhenEvent(mage.ControllerPlaysLand(player, land, ordinal).AfterFirst(), mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}
