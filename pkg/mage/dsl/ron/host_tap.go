package ron

import "github.com/benprew/mage-go/pkg/mage"

type permanentEventPlan struct{ name string }

func compileHostTapTrigger(event, body Node, ctx scope) (optionFactory, error) {
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	subject := event.Args[0]
	if err := arity(subject, 1); err != nil {
		return nil, err
	}
	ref, err := compilePermanentRef(subject.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if ref.kind != permanentHost {
		return nil, unsupported(subject.Args[0], "host tap event reference")
	}
	ctx.permanentEvent = &permanentEventPlan{name: "Trigger_ThatPermanent"}
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		host := mage.BindPermanent(ctx.permanentEvent.name)
		actions := plan.buildWithBindings(instanceBindings{tapped: map[*permanentEventPlan]*mage.PermanentBinding{ctx.permanentEvent: host}})
		trigger := mage.WhenEvent(mage.AttachedBecomesTapped(host), mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}
