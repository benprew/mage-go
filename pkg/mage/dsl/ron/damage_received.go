package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileDamageReceivedTrigger(event, body Node, ctx scope) (optionFactory, error) {
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	permanent := event.Name == "WhenAPermanentIsDealtDamage"
	subject := event.Args[0]
	if permanent {
		if subject.Name != "SinglePermanent" {
			return nil, unsupported(subject, "damage recipient filter")
		}
		if err := arity(subject, 1); err != nil {
			return nil, err
		}
		ref, err := compilePermanentRef(subject.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		if ref.kind != permanentSource || subject.Args[0].Name != "ThisPermanent" {
			return nil, unsupported(subject.Args[0], "damage recipient source")
		}
		ctx.receivedSource = true
	} else {
		if subject.Name != "SinglePlayer" {
			return nil, unsupported(subject, "damage recipient player filter")
		}
		if err := arity(subject, 1); err != nil {
			return nil, err
		}
		player := subject.Args[0]
		if player.Kind != Symbol || player.Name != "You" {
			return nil, unsupported(player, "damage recipient player")
		}
	}
	ctx.damageAmount = &damageAmountPlan{name: "Trigger_AmountOfDamageDealt"}
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		amount := mage.BindNumber(ctx.damageAmount.name)
		source := mage.BindDamageSource("damage event source")
		actions := plan.buildWithBindings(instanceBindings{damageAmounts: map[*damageAmountPlan]*mage.NumberBinding{ctx.damageAmount: amount}})
		var pattern mage.EventPattern = mage.ControllerIsDealtDamage(amount).WithSource(source)
		if permanent {
			pattern = mage.ReceivesDamage(mage.This, amount).WithSource(source)
		}
		trigger := mage.WhenEvent(pattern, mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}
