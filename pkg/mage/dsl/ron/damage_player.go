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
	if ref.kind != permanentSource {
		return nil, unsupported(subject.Args[0], "damage-player source")
	}
	player := event.Args[1]
	if player.Kind != Symbol || (player.Name != "Opponent" && player.Name != "AnyPlayer") {
		return nil, unsupported(player, "damage-player player filter")
	}
	eventPlayerIsOpponent := player.Name == "Opponent"
	ctx.damagePlayer = &damagePlayerPlan{name: "Trigger_ThatPlayer"}
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.Ability {
		player := mage.BindPlayer(ctx.damagePlayer.name)
		actions := plan.buildWithBindings(instanceBindings{damagePlayers: map[*damagePlayerPlan]*mage.PlayerBinding{ctx.damagePlayer: player}})
		var event mage.EventPattern = mage.DealsDamageToPlayer(mage.This, player)
		if eventPlayerIsOpponent {
			event = mage.DealsDamageToOpponent(mage.This, player)
		}
		trigger := mage.WhenEvent(event, mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		actions.effects = nil
		return TriggerA(trigger, actions)
	}, nil
}
