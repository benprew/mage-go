package ron

import "github.com/benprew/mage-go/pkg/mage"

type damagePlayerPlan struct{ name string }

func compileDamagePlayerTrigger(event, body Node, ctx scope) (optionFactory, error) {
	ability, err := compileDamagePlayerAbility(event, body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption { return mage.WithAbility(ability()) }, nil
}

func compileDamagePlayerAbility(event, body Node, ctx scope) (func() mage.Ability, error) {
	if err := arity(event, 2); err != nil {
		return nil, err
	}
	subject := event.Args[0]
	if subject.Name != "SinglePermanent" {
		return nil, unsupported(subject, "damage-player subject")
	}
	if err := arity(subject, 1); err != nil {
		return nil, err
	}
	ref, err := compilePermanentRef(subject.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if ref.kind != permanentSource && ref.kind != permanentHost {
		return nil, unsupported(subject.Args[0], "damage-player source")
	}
	player := event.Args[1]
	controllerOnly := player.Name == "SinglePlayer"
	if controllerOnly {
		if err := arity(player, 1); err != nil {
			return nil, err
		}
		if player.Args[0].Kind != Symbol || player.Args[0].Name != "You" {
			return nil, unsupported(player.Args[0], "damage-player controller filter")
		}
	} else if player.Kind != Symbol || (player.Name != "Opponent" && player.Name != "AnyPlayer") {
		return nil, unsupported(player, "damage-player player filter")
	}
	eventPlayerIsOpponent := player.Name == "Opponent"
	if eventPlayerIsOpponent && ref.kind != permanentSource {
		return nil, unsupported(player, "host damage-player opponent filter")
	}
	ctx.damagePlayer = &damagePlayerPlan{name: "Trigger_ThatPlayer"}
	ctx.damageAmount = &damageAmountPlan{name: "Trigger_AmountOfDamageDealt"}
	if !eventPlayerIsOpponent {
		ctx.permanentEvent = &permanentEventPlan{name: "Trigger_ThatPermanent"}
	}
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.Ability {
		player := mage.BindPlayer(ctx.damagePlayer.name)
		amount := mage.BindNumber(ctx.damageAmount.name)
		bindings := instanceBindings{
			damagePlayers: map[*damagePlayerPlan]*mage.PlayerBinding{ctx.damagePlayer: player},
			damageAmounts: map[*damageAmountPlan]*mage.NumberBinding{ctx.damageAmount: amount},
		}
		pattern := mage.DealsDamageToPlayer(ref.build(bindings), player)
		if controllerOnly {
			pattern = pattern.ToController()
		}
		if ctx.permanentEvent != nil {
			source := mage.BindPermanent(ctx.permanentEvent.name)
			bindings.tapped = map[*permanentEventPlan]*mage.PermanentBinding{ctx.permanentEvent: source}
			pattern = pattern.WithSourcePermanent(source)
		}
		var event = pattern.WithAmount(amount)
		if eventPlayerIsOpponent {
			event = mage.DealsDamageToOpponent(mage.This, player).WithAmount(amount)
		}
		actions := plan.buildWithBindings(bindings)
		trigger := mage.WhenEvent(event, mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		actions.effects = nil
		return TriggerA(trigger, actions)
	}, nil
}
