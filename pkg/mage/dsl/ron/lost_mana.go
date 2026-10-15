package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

type lostManaPlan struct{ name string }

func hasLostManaRead(nodes []Node) bool {
	for _, n := range nodes {
		if n.Name == "TheManaLostThisWay" || hasLostManaRead(n.Args) {
			return true
		}
	}
	return false
}

func lostManaProducer(n Node) bool {
	return n.Name == "LoseUnspentMana" || n.Name == "PlayerAction" && len(n.Args) == 2 && (n.Args[1].Name == "LoseUnspentMana" || n.Args[1].Name == forcedManaAction)
}

func compileLostManaProducer(n Node, remaining []Node, ctx scope) (effectFactory, error) {
	if !ctx.resolving || ctx.drawReplacementBody {
		return nil, problem(n, "saved mana requires an ordinary resolving body")
	}
	player := ctx.player
	action := n
	if n.Name == "PlayerAction" {
		var err error
		player, err = compilePlayerRef(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		action = n.Args[1]
	}
	forced := action.Name == forcedManaAction
	if forced {
		var err error
		player, err = compileForcedManaPlayer(n, ctx)
		if err != nil {
			return nil, err
		}
	} else if action.Kind != Symbol {
		return nil, problem(action, "LoseUnspentMana requires a symbol")
	}
	result := &lostManaPlan{name: "lost mana " + n.Position.String()}
	ctx.lostMana = result
	body, err := compileActionSequence(remaining, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		bindings.lostMana = maps.Clone(bindings.lostMana)
		if bindings.lostMana == nil {
			bindings.lostMana = make(map[*lostManaPlan]*mage.LostManaBinding)
		}
		binding := mage.BindLostMana(result.name)
		bindings.lostMana[result] = binding
		effects := make([]mage.Effect, 0, 1+len(body))
		ref := player.build(bindings)
		if forced {
			saved := mage.BindPlayer("forced mana player " + n.Position.String())
			effects = append(effects, mage.SnapshotPlayer(ref, saved))
			ref = mage.FromPlayer(saved)
			effects = append(effects, mage.ActivateManaAbilityOfEach(ref, mage.IsLand))
		}
		effects = append(effects, mage.LoseUnspentManaInto(ref, binding))
		for _, build := range body {
			effects = append(effects, build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}

func compileLostManaConsumer(n Node, ctx scope) ([]effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	if n.Args[0].Kind != Symbol || ctx.lostMana == nil {
		return nil, problem(n.Args[0], "unbound or incompatible lost mana result")
	}
	if !ctx.resolving || ctx.drawReplacementBody {
		return nil, problem(n, "saved mana requires an ordinary resolving body")
	}
	return []effectFactory{func(bindings instanceBindings) mage.Effect {
		return mage.AddLostManaFor(ctx.player.build(bindings), bindings.lostMana[ctx.lostMana])
	}}, nil
}
