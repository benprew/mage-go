package ron

import "github.com/benprew/mage-go/pkg/mage"

type lastCounterEventPlan struct{ actorName string }

func compileLastCounterTrigger(event, body Node, ctx scope) (optionFactory, error) {
	if err := arity(event, 3); err != nil {
		return nil, err
	}
	players := event.Args[0]
	if players.Name != "SinglePlayer" {
		return nil, unsupported(players, "last-counter player filter")
	}
	if err := arity(players, 1); err != nil {
		return nil, err
	}
	if players.Args[0].Kind != Symbol || players.Args[0].Name != "You" {
		return nil, unsupported(players.Args[0], "last-counter player reference")
	}
	counter, err := compilePermanentCounterType(event.Args[1])
	if err != nil {
		return nil, err
	}
	subject := event.Args[2]
	if subject.Name != "SinglePermanent" {
		return nil, unsupported(subject, "last-counter permanent filter")
	}
	if err := arity(subject, 1); err != nil {
		return nil, err
	}
	if !ctx.permanent || subject.Args[0].Kind != Symbol || subject.Args[0].Name != "ThisPermanent" {
		return nil, unsupported(subject.Args[0], "last-counter source reference")
	}
	ctx.lastCounterEvent = &lastCounterEventPlan{actorName: "counter remover"}
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		actor := mage.BindPlayer(ctx.lastCounterEvent.actorName)
		actions := plan.buildWithBindings(instanceBindings{counterRemovers: map[*lastCounterEventPlan]*mage.PlayerBinding{ctx.lastCounterEvent: actor}})
		trigger := mage.WhenEvent(mage.ControllerRemovesLastCounter(counter, actor), mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}
