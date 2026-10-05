package ron

import "github.com/benprew/mage-go/pkg/mage"

type damagePlayerPlan struct{ name string }

func compileDamagePlayerTrigger(event, body Node, ctx scope) (optionFactory, error) {
	if err := arity(event, 2); err != nil {
		return nil, err
	}
	subject := event.Args[0]
	if subject.Name != "SinglePermanent" {
		return nil, unsupported(subject, "opponent-damage subject")
	}
	if err := arity(subject, 1); err != nil {
		return nil, err
	}
	ref, err := compilePermanentRef(subject.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if ref.kind != permanentSource {
		return nil, unsupported(subject.Args[0], "opponent-damage source")
	}
	player := event.Args[1]
	if player.Kind != Symbol || player.Name != "Opponent" {
		return nil, unsupported(player, "opponent-damage player filter")
	}
	ctx.damagePlayer = &damagePlayerPlan{name: "Trigger_ThatPlayer"}
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		player := mage.BindPlayer(ctx.damagePlayer.name)
		actions := plan.buildWithBindings(instanceBindings{damagePlayers: map[*damagePlayerPlan]*mage.PlayerBinding{ctx.damagePlayer: player}})
		trigger := mage.WhenEvent(mage.DealsDamageToOpponent(mage.This, player), mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}
