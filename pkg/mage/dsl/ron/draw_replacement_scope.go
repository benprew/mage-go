package ron

import "github.com/benprew/mage-go/pkg/mage"

type drawReplacementPlan struct{ name string }

func compileDrawReplacement(n Node, ctx scope) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if !ctx.permanent {
		return nil, unsupported(n, "draw replacement source")
	}
	event := n.Args[0]
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	window := mage.ControllerDrawStep
	player := "You"
	switch event.Name {
	case "PlayerWouldDrawDuringTheirDrawStep":
	case "APlayerWouldDrawExceptFirstDrawStepDraw":
		window = mage.ExceptFirstDrawStepDraw
		player = "AnyPlayer"
	default:
		return nil, unsupported(event, "draw replacement event")
	}
	if event.Args[0].Kind != Symbol || event.Args[0].Name != player {
		return nil, unsupported(event.Args[0], "draw replacement players")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) == 0 {
		return nil, problem(n.Args[1], "supply a draw replacement body")
	}
	plan := &drawReplacementPlan{name: "drawing player " + event.Position.String()}
	ctx.drawReplacement = plan
	actions, err := compileActionSequence(operations, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		bindings := instanceBindings{drawReplacements: map[*drawReplacementPlan]*mage.PlayerBinding{plan: mage.BindPlayer(plan.name)}}
		effects := make([]mage.Effect, len(actions))
		for i, build := range actions {
			effects[i] = build(bindings)
		}
		return mage.WithAbility(mage.Static(mage.ReplaceDraw(window, bindings.drawReplacements[plan], mage.Sequence(effects...))))
	}, nil
}
