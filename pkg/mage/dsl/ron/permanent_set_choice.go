package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

type permanentSetChoicePlan struct{ name string }

func compilePermanentSetChoice(n Node, remaining []Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if n.Args[0].Kind != Symbol || n.Args[0].Name != "AnyPlayer" {
		return nil, unsupported(n.Args[0], "plural choice player collection")
	}
	action := n.Args[1]
	if err := arity(action, 3); err != nil {
		return nil, err
	}
	limit := action.Args[0]
	if err := arity(limit, 1); err != nil {
		return nil, err
	}
	if limit.Name != "Integer" || limit.Args[0].Kind != IntegerNode || limit.Args[0].Integer < 0 {
		return nil, problem(limit, "choice bound must be a nonnegative Integer")
	}
	subjects, err := compileContextualCollection(action.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	count := action.Args[2]
	if count.Name != "NumberOfPermanentsTappedThisWayByPlayer" {
		return nil, unsupported(count, "plural choice repetition count")
	}
	if err := arity(count, 1); err != nil {
		return nil, err
	}
	if count.Args[0].Kind != Symbol || count.Args[0].Name != "EachablePlayer" {
		return nil, unsupported(count.Args[0], "plural choice count player")
	}
	if ctx.playerTap == nil {
		return nil, problem(count, "unbound per-player tap result")
	}
	plan := &permanentSetChoicePlan{name: "kept choices " + n.Position.String()}
	ctx.permanentSetChoice = plan
	body, err := compileActionSequence(remaining, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		bindings.permanentSetChoices = maps.Clone(bindings.permanentSetChoices)
		if bindings.permanentSetChoices == nil {
			bindings.permanentSetChoices = make(map[*permanentSetChoicePlan]*mage.PermanentSetBinding)
		}
		result := mage.BindPermanentSet(plan.name)
		bindings.permanentSetChoices[plan] = result
		effects := make([]mage.Effect, 0, 1+len(body))
		effects = append(effects, mage.ChooseUpToPermanentsEach(mage.EachPlayer, subjects(bindings), mage.Fixed(limit.Args[0].Integer), bindings.playerTaps[ctx.playerTap], result))
		for _, build := range body {
			effects = append(effects, build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}
