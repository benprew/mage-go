package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

type permanentChoicePlan struct{ name string }

type permanentChoiceHistoryPlan struct{ choices []*permanentChoicePlan }

func compilePermanentChoice(n Node, remaining []Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	subjects, err := compileContextualCollection(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	plan := &permanentChoicePlan{name: "chosen permanent " + n.Position.String()}
	player := ctx.player
	ctx.chosenPermanent = plan
	return compilePermanentChoiceContinuation(remaining, ctx, plan, player, subjects)
}

func compilePermanentChoiceContinuation(remaining []Node, ctx scope, plan *permanentChoicePlan, player playerReferencePlan, subjects permanentSetFactory) (effectFactory, error) {
	var preceding []*permanentChoicePlan
	if ctx.permanentChoices != nil {
		preceding = ctx.permanentChoices.choices
	}
	ctx.permanentChoices = &permanentChoiceHistoryPlan{choices: append(append([]*permanentChoicePlan(nil), preceding...), plan)}
	body, err := compileActionSequence(remaining, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		binding := mage.BindPermanentChoice(plan.name)
		bindings.chosenPermanents = maps.Clone(bindings.chosenPermanents)
		if bindings.chosenPermanents == nil {
			bindings.chosenPermanents = make(map[*permanentChoicePlan]*mage.PermanentChoiceBinding)
		}
		bindings.chosenPermanents[plan] = binding
		effects := make([]mage.Effect, 0, 1+len(body))
		effects = append(effects, mage.ChoosePermanentForPlayer(player.build(bindings), subjects(bindings), binding))
		for _, build := range body {
			effects = append(effects, build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}
