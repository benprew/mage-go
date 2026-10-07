package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileUpkeepPlayerCondition(n Node, ctx scope) (optionFactory, error) {
	event := n.Args[0]
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	players := event.Args[0]
	own := false
	if players.Kind != Symbol || players.Name != "AnyPlayer" {
		if players.Name != "SinglePlayer" {
			return nil, unsupported(players, "upkeep condition player filter")
		}
		if err := arity(players, 1); err != nil {
			return nil, err
		}
		if players.Args[0].Kind != Symbol || players.Args[0].Name != "You" {
			return nil, unsupported(players.Args[0], "upkeep condition player reference")
		}
		own = true
	}
	ctx.stepEvent = &stepEventPlan{playerName: "Trigger_ThatPlayer"}
	condition, err := compileUpkeepCondition(n.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	plan, err := compileActions(n.Args[2], ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		player := mage.BindPlayer(ctx.stepEvent.playerName)
		bindings := instanceBindings{steps: map[*stepEventPlan]*mage.PlayerBinding{ctx.stepEvent: player}}
		actions := plan.buildWithBindings(bindings)
		step := mage.EachUpkeep
		if own {
			step = mage.YourUpkeep
		}
		trigger := mage.WhenEvent(mage.TurnStepBegins(step, player), mage.Sequence(actions.effects...), mage.WithInterveningIf(condition(bindings))).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}

func compileUpkeepCondition(n Node, ctx scope) (func(instanceBindings) mage.StateCondition, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	selector := n.Args[1]
	switch n.Name {
	case "PlayerPassesFilter":
		player := n.Args[0]
		if player.Kind != Symbol || (player.Name != "You" && player.Name != "Trigger_ThatPlayer") {
			return nil, unsupported(player, "upkeep condition player reference")
		}
		ref, err := compilePlayerRef(player, ctx)
		if err != nil {
			return nil, err
		}
		if selector.Name != "ControlsA" {
			return nil, unsupported(selector, "upkeep player condition")
		}
		if err := arity(selector, 1); err != nil {
			return nil, err
		}
		filter, err := compileStaticConditionFilter(selector.Args[0])
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.StateCondition {
			return mage.PlayerControls(ref.build(bindings), filter)
		}, nil
	case "APlayerPassesFilter":
		players, err := symbol(n.Args[0])
		if err != nil {
			return nil, err
		}
		if players == "Opponent" && selector.Name == "ControlsMorePermanentThanPlayer" {
			if err := arity(selector, 2); err != nil {
				return nil, err
			}
			if selector.Args[0].Kind != Symbol || selector.Args[0].Name != "You" {
				return nil, unsupported(selector.Args[0], "upkeep count comparison player")
			}
			filter, err := compileStaticConditionFilter(selector.Args[1])
			if err != nil {
				return nil, err
			}
			return func(instanceBindings) mage.StateCondition { return mage.OpponentControlsMoreThanYou(filter) }, nil
		}
		if players == "AnyPlayer" && selector.Name == "SinglePlayer" {
			if err := arity(selector, 1); err != nil {
				return nil, err
			}
			if selector.Args[0].Kind != Symbol || selector.Args[0].Name != "ThePlayerWithTheMostLife" {
				return nil, unsupported(selector.Args[0], "upkeep highest-life player")
			}
			return func(instanceBindings) mage.StateCondition { return mage.HasUniqueHighestLife() }, nil
		}
	}
	return nil, unsupported(n, "upkeep player condition")
}
