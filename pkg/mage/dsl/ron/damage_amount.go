package ron

import "github.com/benprew/mage-go/pkg/mage"

type damageAmountPlan struct{ name string }

func compileDamageAmountTrigger(event, body Node, ctx scope) (optionFactory, error) {
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	subject := event.Args[0]
	if subject.Name != "SinglePermanent" {
		return nil, unsupported(subject, "damage event subject")
	}
	if err := arity(subject, 1); err != nil {
		return nil, err
	}
	ref, err := compilePermanentRef(subject.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if ref.kind != permanentSource && ref.kind != permanentHost {
		return nil, unsupported(subject.Args[0], "damage event source")
	}
	ctx.damageAmount = &damageAmountPlan{name: "Trigger_AmountOfDamageDealt"}
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		amount := mage.BindNumber(ctx.damageAmount.name)
		actions := plan.buildWithBindings(instanceBindings{damageAmounts: map[*damageAmountPlan]*mage.NumberBinding{ctx.damageAmount: amount}})
		trigger := mage.WhenEvent(mage.DealsDamage(ref.build(instanceBindings{}), amount), mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}
