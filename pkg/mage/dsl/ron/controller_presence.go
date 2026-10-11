package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileControllerPresenceTrigger(event, body Node, ctx scope) (optionFactory, error) {
	if err := arity(event, 2); err != nil {
		return nil, err
	}
	player := event.Args[0]
	if player.Name != "SinglePlayer" {
		return nil, unsupported(player, "controller-presence player")
	}
	if err := arity(player, 1); err != nil {
		return nil, err
	}
	if player.Args[0].Kind != Symbol || player.Args[0].Name != "You" {
		return nil, unsupported(player.Args[0], "controller-presence player reference")
	}
	if err := validatePresenceFilter(event.Args[1]); err != nil {
		return nil, err
	}
	filter, err := compileFilter(event.Args[1])
	if err != nil {
		return nil, err
	}
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		actions := plan.build()
		trigger := mage.WhenState(mage.Controls(filter), mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}

func validatePresenceFilter(n Node) error {
	if n.Name != "And" && n.Name != "Or" {
		return nil
	}
	if err := arity(n, 1); err != nil {
		return err
	}
	parts, err := list(n.Args[0])
	if err != nil {
		return err
	}
	if len(parts) == 0 {
		return problem(n, "presence filter requires a nonempty group")
	}
	for _, part := range parts {
		if err := validatePresenceFilter(part); err != nil {
			return err
		}
	}
	return nil
}
