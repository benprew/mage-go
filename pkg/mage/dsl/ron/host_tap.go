package ron

import "github.com/benprew/mage-go/pkg/mage"

type permanentEventPlan struct {
	name       string
	referenced bool
}

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
	if ref.kind == permanentSource {
		ctx.sourceTapped = true
		plan, err := compileTapTriggerBody(body, ctx)
		if err != nil {
			return nil, err
		}
		return func() mage.CardOption {
			return mage.WithAbility(plan(mage.BecomesTapped(mage.This), instanceBindings{}))
		}, nil
	}
	if ref.kind != permanentHost {
		return nil, unsupported(subject.Args[0], "host tap event reference")
	}
	ctx.permanentEvent = &permanentEventPlan{name: "Trigger_ThatPermanent"}
	plan, err := compileTapTriggerBody(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		host := mage.BindPermanent(ctx.permanentEvent.name)
		bindings := instanceBindings{tapped: map[*permanentEventPlan]*mage.PermanentBinding{ctx.permanentEvent: host}}
		return mage.WithAbility(plan(mage.AttachedBecomesTapped(host), bindings))
	}, nil
}

func compileTapTriggerBody(body Node, ctx scope) (func(mage.EventPattern, instanceBindings) *mage.GenericTriggered, error) {
	if body.Name == "Modal_ChooseOne" {
		plans, err := compileChooseOneModes(body, ctx)
		if err != nil {
			return nil, err
		}
		return func(pattern mage.EventPattern, bindings instanceBindings) *mage.GenericTriggered {
			trigger := mage.WhenEvent(pattern, nil).(*mage.GenericTriggered)
			return trigger.WithModes(buildChooseOneModes(plans, bindings)...)
		}, nil
	}
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func(pattern mage.EventPattern, bindings instanceBindings) *mage.GenericTriggered {
		actions := plan.buildWithBindings(bindings)
		trigger := mage.WhenEvent(pattern, mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		actions.effects = nil
		return TriggerA(trigger, actions)
	}, nil
}
