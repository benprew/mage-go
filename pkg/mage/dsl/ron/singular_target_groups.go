package ron

import (
	"strconv"

	"github.com/benprew/mage-go/pkg/mage"
)

func compileSingularPermanentTarget(n Node, ctx scope, name string) (targetFactory, error) {
	count := 1
	chosen := n.Name == "TargetPermanentOfAPlayersChoice" || n.Name == "TargetPermanentOfAPlayersChoiceTheyControl"
	if chosen {
		count = 2
	}
	if err := arity(n, count); err != nil {
		return nil, err
	}
	if chosen && (n.Args[1].Kind != Symbol || n.Args[1].Name != "Opponent") {
		return nil, unsupported(n.Args[1], "permanent target chooser")
	}
	filter, options, err := compilePlayerRelativeTargetFilter(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	you, opponent := singularTargetControllers(n.Args[0])
	if n.Name == "TargetPermanentOfAPlayersChoiceTheyControl" {
		opponent = true
	}
	if you && opponent {
		return nil, problem(n.Args[0], "conflicting target controller restrictions")
	}
	return func(bindings instanceBindings) mage.Target {
		built := options(bindings)
		if chosen {
			built = append(built, mage.TargetChosenByOpponent())
		}
		if n.Name == "TargetPermanentOfAPlayersChoiceTheyControl" {
			built = append(built, mage.TargetOpponentControls())
		}
		return mage.DeclarePermanentTarget(ctx.targetNamespace+name, filter, built...)
	}, nil
}

func compileSingularPermanentGroups(n Node, targets []Node, ctx scope) (actionPlan, error) {
	var plan actionPlan
	for _, target := range targets {
		if !isSingularPermanentTarget(target) {
			return plan, unsupported(target, "singular permanent target group")
		}
	}
	ctx.permanentTarget = nil
	ctx.permanentGroups = make(map[string]*permanentTargetPlan, len(targets))
	builds := make([]targetFactory, len(targets))
	groups := make([]*permanentTargetPlan, len(targets))
	for i, target := range targets {
		name := "Ref_TargetPermanent" + strconv.Itoa(i+1)
		build, err := compileSingularPermanentTarget(target, ctx, name)
		if err != nil {
			return plan, err
		}
		groups[i] = &permanentTargetPlan{name: name}
		ctx.permanentGroups[name] = groups[i]
		builds[i] = build
	}
	plan, err := compileActions(n.Args[1], ctx)
	if err != nil {
		return plan, err
	}
	if len(plan.targets) != 0 {
		return plan, problem(n, "nested target declarations are not supported")
	}
	plan.targets = builds
	plan.permanentGroups = groups
	return plan, nil
}

func singularTargetControllers(n Node) (you, opponent bool) {
	switch n.Name {
	case "And":
		for _, child := range n.Args[0].Args {
			childYou, childOpponent := singularTargetControllers(child)
			you, opponent = you || childYou, opponent || childOpponent
		}
	case "ControlledByPlayer":
		you = n.Args[0].Kind == Symbol && n.Args[0].Name == "You"
	case "ControlledByAPlayer":
		player := n.Args[0]
		opponent = player.Kind == Symbol && player.Name == "Opponent"
		you = player.Name == "SinglePlayer" && player.Args[0].Kind == Symbol && player.Args[0].Name == "You"
	}
	return you, opponent
}

func isSingularPermanentTarget(n Node) bool {
	return n.Name == "TargetPermanent" || n.Name == "TargetPermanentOfAPlayersChoice" || n.Name == "TargetPermanentOfAPlayersChoiceTheyControl"
}
