package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

type playerDiscardPlan struct{ name string }

func compilePlayerDiscardResults(n Node, remaining []Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if n.Args[0].Kind != Symbol || n.Args[0].Name != "AnyPlayer" {
		return nil, unsupported(n.Args[0], "discard player collection")
	}
	action := n.Args[1]
	if err := arity(action, 1); err != nil {
		return nil, err
	}
	maximum := action.Args[0]
	if maximum.Name != "Integer" || maximum.Kind != Tuple || len(maximum.Args) != 1 || maximum.Args[0].Kind != IntegerNode || maximum.Args[0].Integer < 0 {
		return nil, problem(maximum, "optional discard requires a nonnegative integer maximum")
	}
	amount := maximum.Args[0].Integer
	plan := &playerDiscardPlan{name: "discarded counts " + n.Position.String()}
	ctx.playerDiscard = plan
	body, err := compileActionSequence(remaining, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		bindings.playerDiscards = maps.Clone(bindings.playerDiscards)
		if bindings.playerDiscards == nil {
			bindings.playerDiscards = make(map[*playerDiscardPlan]*mage.PlayerNumbersBinding)
		}
		result := mage.BindPlayerNumbers(plan.name)
		bindings.playerDiscards[plan] = result
		effects := make([]mage.Effect, 0, 1+len(body))
		effects = append(effects, (mage.DiscardUpToInstruction{Subjects: mage.EachPlayer, Maximum: mage.Fixed(amount), Result: result}).Effect())
		for _, build := range body {
			effects = append(effects, build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}
func compilePlayerDiscardValue(n Node, ctx scope) (valueFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	if ctx.playerDiscard == nil {
		return nil, problem(n, "unbound per-player discard result")
	}
	player, err := compilePlayerRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.ResolutionValue {
		return mage.PlayerNumber(bindings.playerDiscards[ctx.playerDiscard], player.build(bindings))
	}, nil
}
