package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

type paidPlayersPlan struct{ name string }

func compileOptionalAntePlayers(n Node, remaining []Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "optional ante players require an ordinary resolving body")
	}
	if n.Args[0].Kind != Symbol || n.Args[0].Name != "AnyPlayer" {
		return nil, unsupported(n.Args[0], "optional payment player collection")
	}
	if n.Args[1].Kind != Symbol || n.Args[1].Name != "AnteTopCardOfLibrary" {
		return nil, unsupported(n.Args[1], "optional player collection cost")
	}
	plan := &paidPlayersPlan{name: "paid players " + n.Position.String()}
	ctx.paidPlayers = plan
	body, err := compileActionSequence(remaining, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		bindings.paidPlayers = maps.Clone(bindings.paidPlayers)
		if bindings.paidPlayers == nil {
			bindings.paidPlayers = make(map[*paidPlayersPlan]*mage.PlayerSetBinding)
		}
		saved := mage.BindPlayerSet(plan.name)
		bindings.paidPlayers[plan] = saved
		effects := make([]mage.Effect, 0, 1+len(body))
		effects = append(effects, mage.MayAnteTopEach(mage.EachPlayer, saved))
		for _, build := range body {
			effects = append(effects, build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}

func compilePaidPlayersAction(n Node, ctx scope) ([]effectFactory, error) {
	if n.Args[0].Kind != Symbol || ctx.paidPlayers == nil || !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n.Args[0], "PaidCost requires a preceding local player payment")
	}
	action := n.Args[1]
	if action.Name != "SetLifeTotal" {
		return nil, unsupported(action, "paid-player action")
	}
	if err := arity(action, 1); err != nil {
		return nil, err
	}
	value := action.Args[0]
	if value.Name != "Integer" || value.Kind != Tuple || len(value.Args) != 1 || value.Args[0].Kind != IntegerNode {
		return nil, problem(value, "paid-player life total requires a fixed integer")
	}
	return []effectFactory{func(bindings instanceBindings) mage.Effect {
		return mage.SetLifeTotalEach(mage.FromPlayerSet(bindings.paidPlayers[ctx.paidPlayers]), mage.Fixed(value.Args[0].Integer))
	}}, nil
}
