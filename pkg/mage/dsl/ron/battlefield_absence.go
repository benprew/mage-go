package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileBattlefieldAbsenceTrigger(event, body Node, ctx scope) (optionFactory, error) {
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	if err := validatePresenceFilter(event.Args[0]); err != nil {
		return nil, err
	}
	filter, err := compileFilter(event.Args[0])
	if err != nil {
		return nil, err
	}
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		actions := plan.build()
		trigger := mage.WhenState(mage.BattlefieldHasNone(filter), mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}
