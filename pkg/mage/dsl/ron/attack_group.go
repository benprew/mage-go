package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileAttackGroupTrigger(event, body Node, ctx scope) (optionFactory, error) {
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	filter := event.Args[0]
	controlled := false
	if filter.Name == "And" {
		if err := arity(filter, 1); err != nil {
			return nil, err
		}
		parts, err := list(filter.Args[0])
		if err != nil {
			return nil, err
		}
		if len(parts) != 2 {
			return nil, problem(filter, "attack group requires a creature type and controller")
		}
		creature, controller := parts[0], parts[1]
		if controller.Name == "IsCardtype" {
			creature, controller = controller, creature
		}
		if controller.Name != "ControlledByAPlayer" {
			return nil, unsupported(controller, "attack group controller")
		}
		if err := arity(controller, 1); err != nil {
			return nil, err
		}
		player := controller.Args[0]
		if player.Name != "SinglePlayer" {
			return nil, unsupported(player, "attack group player")
		}
		if err := arity(player, 1); err != nil {
			return nil, err
		}
		if player.Args[0].Kind != Symbol || player.Args[0].Name != "You" {
			return nil, unsupported(player.Args[0], "attack group player")
		}
		filter = creature
		controlled = true
	}
	if filter.Name != "IsCardtype" {
		return nil, unsupported(filter, "attack group creature filter")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	if filter.Args[0].Kind != Symbol || filter.Args[0].Name != "Creature" {
		return nil, unsupported(filter.Args[0], "attack group creature type")
	}
	ctx.attackGroup = &sourceCombatPlayerPlan{name: "attack group defender " + event.Position.String()}
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		defender := mage.BindPlayer(ctx.attackGroup.name)
		actions := plan.buildWithBindings(instanceBindings{defenders: map[*sourceCombatPlayerPlan]*mage.PlayerBinding{ctx.attackGroup: defender}})
		pattern := mage.OneOrMoreCreaturesAttack().WithDefendingPlayer(defender)
		if controlled {
			pattern = mage.OneOrMoreCreaturesYouControlAttack().WithDefendingPlayer(defender)
		}
		trigger := mage.WhenEvent(pattern, mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}
