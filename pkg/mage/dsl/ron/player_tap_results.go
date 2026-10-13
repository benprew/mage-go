package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

type playerTapPlan struct{ name string }

func compilePlayerTapResults(n Node, remaining []Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if n.Args[0].Kind != Symbol || n.Args[0].Name != "AnyPlayer" {
		return nil, unsupported(n.Args[0], "tap player collection")
	}
	action := n.Args[1]
	if err := arity(action, 1); err != nil {
		return nil, err
	}
	filter, controllers, err := removeTapActionController(action.Args[0])
	if err != nil {
		return nil, err
	}
	if controllers != 1 {
		return nil, problem(action, "optional tap requires one conjunctive ActionPlayer controller filter")
	}
	subjects, err := compileContextualCollection(filter, ctx)
	if err != nil {
		return nil, err
	}
	plan := &playerTapPlan{name: "tapped counts " + n.Position.String()}
	ctx.playerTap = plan
	body, err := compileActionSequence(remaining, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		bindings.playerTaps = maps.Clone(bindings.playerTaps)
		if bindings.playerTaps == nil {
			bindings.playerTaps = make(map[*playerTapPlan]*mage.PlayerNumbersBinding)
		}
		result := mage.BindPlayerNumbers(plan.name)
		bindings.playerTaps[plan] = result
		effects := make([]mage.Effect, 0, 1+len(body))
		effects = append(effects, mage.TapAnyNumberEach(mage.EachPlayer, subjects(bindings), result))
		for _, build := range body {
			effects = append(effects, build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}

func removeTapActionController(n Node) (Node, int, error) {
	if n.Name == "ControlledByAPlayer" {
		if err := arity(n, 1); err != nil {
			return Node{}, 0, err
		}
		player := n.Args[0]
		if player.Name != "SinglePlayer" {
			return Node{}, 0, unsupported(player, "tap controller collection")
		}
		if err := arity(player, 1); err != nil {
			return Node{}, 0, err
		}
		if player.Args[0].Kind != Symbol || player.Args[0].Name != "ActionPlayer" {
			return Node{}, 0, unsupported(player.Args[0], "tap controller")
		}
		return Node{Kind: Symbol, Name: "AnyPermanent", Position: n.Position}, 1, nil
	}
	if n.Name != "And" {
		return n, 0, nil
	}
	if err := arity(n, 1); err != nil {
		return Node{}, 0, err
	}
	parts, err := list(n.Args[0])
	if err != nil {
		return Node{}, 0, err
	}
	result := make([]Node, 0, len(parts))
	count := 0
	for _, part := range parts {
		child, found, childErr := removeTapActionController(part)
		if childErr != nil {
			return Node{}, 0, childErr
		}
		result = append(result, child)
		count += found
	}
	n.Args = []Node{{Kind: List, Args: result, Position: n.Args[0].Position}}
	return n, count, nil
}
func compilePlayerTapValue(n Node, ctx scope) (valueFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	if ctx.playerTap == nil {
		return nil, problem(n, "unbound per-player tap result")
	}
	player, err := compilePlayerRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.ResolutionValue {
		return mage.PlayerNumber(bindings.playerTaps[ctx.playerTap], player.build(bindings))
	}, nil
}
