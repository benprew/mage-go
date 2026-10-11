package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

func compileAPNAPPermanentChoices(n Node, remaining []Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	choices, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	if len(choices) == 0 {
		return nil, problem(n, "permanent choice group must not be empty")
	}
	plans := make([]*permanentChoicePlan, len(choices))
	players := make([]playerReferencePlan, len(choices))
	subjects := make([]permanentSetFactory, len(choices))
	for i, choice := range choices {
		if choice.Name != "PlayerAction" {
			return nil, unsupported(choice, "grouped permanent choice")
		}
		if err := arity(choice, 2); err != nil {
			return nil, err
		}
		players[i], err = compilePlayerRef(choice.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		action := choice.Args[1]
		if action.Name != "ChooseAPermanent" {
			return nil, unsupported(action, "grouped permanent choice")
		}
		if err := arity(action, 1); err != nil {
			return nil, err
		}
		local := ctx
		local.player = players[i]
		subjects[i], err = compileContextualCollection(action.Args[0], local)
		if err != nil {
			return nil, err
		}
		plans[i] = &permanentChoicePlan{name: "grouped permanent choice " + choice.Position.String()}
	}
	var preceding []*permanentChoicePlan
	if ctx.permanentChoices != nil {
		preceding = ctx.permanentChoices.choices
	}
	ctx.permanentChoices = &permanentChoiceHistoryPlan{choices: append(append([]*permanentChoicePlan(nil), preceding...), plans...)}
	body, err := compileActionSequence(remaining, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		bindings.chosenPermanents = maps.Clone(bindings.chosenPermanents)
		if bindings.chosenPermanents == nil {
			bindings.chosenPermanents = make(map[*permanentChoicePlan]*mage.PermanentChoiceBinding)
		}
		requests := make([]mage.PermanentChoiceRequest, len(plans))
		for i, plan := range plans {
			result := mage.BindPermanentChoice(plan.name)
			bindings.chosenPermanents[plan] = result
			requests[i] = mage.PermanentChoiceRequest{Player: players[i].build(bindings), Subjects: subjects[i](bindings), Result: result}
		}
		effects := make([]mage.Effect, 0, 1+len(body))
		effects = append(effects, mage.ChoosePermanentsAPNAP(requests...))
		for _, build := range body {
			effects = append(effects, build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}
