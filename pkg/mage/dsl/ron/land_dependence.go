package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileLandDependenceFilter(n Node) (mage.PermanentFilter, error) {
	if n.Name != "IsLandType" {
		return mage.PermanentFilter{}, unsupported(n, "land dependence filter")
	}
	if err := arity(n, 1); err != nil {
		return mage.PermanentFilter{}, err
	}
	subtype, err := symbol(n.Args[0])
	if err != nil {
		return mage.PermanentFilter{}, err
	}
	switch subtype {
	case "Plains", "Island", "Swamp", "Mountain", "Forest", "Desert":
	default:
		return mage.PermanentFilter{}, unsupported(n.Args[0], "land dependence subtype")
	}
	return compileFilter(n)
}

func compileDefendingLandRule(n, operation Node, ctx scope) (optionFactory, error) {
	subject, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if subject.kind != permanentSource {
		return nil, unsupported(n.Args[0], "defending land source")
	}
	if err := arity(operation, 1); err != nil {
		return nil, err
	}
	condition := operation.Args[0]
	if condition.Name != "PlayerPassesFilter" {
		return nil, unsupported(condition, "defending land condition")
	}
	if err := arity(condition, 2); err != nil {
		return nil, err
	}
	if condition.Args[0].Kind != Symbol || condition.Args[0].Name != "DefendingPlayer" {
		return nil, unsupported(condition.Args[0], "defending land player")
	}
	controls := condition.Args[1]
	if controls.Name != "ControlsA" {
		return nil, unsupported(controls, "defending land control")
	}
	if err := arity(controls, 1); err != nil {
		return nil, err
	}
	filter, err := compileLandDependenceFilter(controls.Args[0])
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.Static(mage.CanAttackOnlyIfDefendingPlayerControls(filter)))
	}, nil
}

func compileNoLandTrigger(event, body Node, ctx scope) (optionFactory, error) {
	if err := arity(event, 2); err != nil {
		return nil, err
	}
	player := event.Args[0]
	if player.Name != "SinglePlayer" {
		return nil, unsupported(player, "no-land trigger player")
	}
	if err := arity(player, 1); err != nil {
		return nil, err
	}
	if player.Args[0].Kind != Symbol || player.Args[0].Name != "You" {
		return nil, unsupported(player.Args[0], "no-land trigger player reference")
	}
	filter, err := compileLandDependenceFilter(event.Args[1])
	if err != nil {
		return nil, err
	}
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		actions := plan.build()
		trigger := mage.WhenState(mage.ControlsNone(filter), mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}
