package ron

import "github.com/benprew/mage-go/pkg/mage"

type drawReplacementPlan struct{ name string }

type scopedDrawReplacementPlan struct {
	window  mage.DrawReplacementWindow
	player  *drawReplacementPlan
	actions []effectFactory
}

func (p scopedDrawReplacementPlan) build() mage.CardOption {
	bindings := instanceBindings{drawReplacements: map[*drawReplacementPlan]*mage.PlayerBinding{p.player: mage.BindPlayer(p.player.name)}}
	effects := make([]mage.Effect, len(p.actions))
	for i, build := range p.actions {
		effects[i] = build(bindings)
	}
	return mage.WithAbility(mage.Static(mage.ReplaceDraw(p.window, bindings.drawReplacements[p.player], mage.Sequence(effects...))))
}

func compileDrawReplacement(n Node, ctx scope) (optionFactory, error) {
	plan, err := compileScopedDrawReplacementPlan(n, ctx)
	if err != nil {
		return nil, err
	}
	return plan.build, nil
}

func compileScopedDrawReplacementPlan(n Node, ctx scope) (*scopedDrawReplacementPlan, error) {
	if n.Name != "ReplaceWouldDraw" {
		return nil, unsupported(n, "draw replacement")
	}
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if !ctx.permanent || ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
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
	ctx.drawReplacementBody = true
	actions, err := compileActionSequence(operations, ctx)
	if err != nil {
		return nil, err
	}
	return &scopedDrawReplacementPlan{window: window, player: plan, actions: actions}, nil
}
