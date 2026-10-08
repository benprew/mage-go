package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

type playerChoicePlan struct{ name string }
type playerSetFactory func(instanceBindings) mage.PlayerSet

func compileResolutionPlayerSet(n Node, ctx scope) (playerSetFactory, error) {
	if n.Kind == Symbol {
		switch n.Name {
		case "AnyPlayer":
			return func(instanceBindings) mage.PlayerSet { return mage.EachPlayer }, nil
		case "Opponent":
			player := ctx.player
			return func(bindings instanceBindings) mage.PlayerSet { return mage.OpponentsOf(player.build(bindings)) }, nil
		}
	}
	if n.Name == "CastASpellThisTurn" {
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		filter := n.Args[0]
		if err := arity(filter, 1); err != nil {
			return nil, err
		}
		if filter.Name == "IsCardtype" && filter.Args[0].Kind == Symbol && filter.Args[0].Name == "Sorcery" {
			return func(instanceBindings) mage.PlayerSet { return mage.PlayersWhoCastSorceryThisTurn() }, nil
		}
	}
	return nil, unsupported(n, "resolution player filter")
}

func compilePlayerChoice(n Node, remaining []Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	subjects, err := compileResolutionPlayerSet(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	plan := &playerChoicePlan{name: "chosen player " + n.Position.String()}
	player := ctx.player
	ctx.chosenPlayer = plan
	body, err := compileActionSequence(remaining, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		binding := mage.BindPlayer(plan.name)
		bindings.chosenPlayers = maps.Clone(bindings.chosenPlayers)
		if bindings.chosenPlayers == nil {
			bindings.chosenPlayers = make(map[*playerChoicePlan]*mage.PlayerBinding)
		}
		bindings.chosenPlayers[plan] = binding
		effects := make([]mage.Effect, 0, 1+len(body))
		effects = append(effects, mage.ChoosePlayer(player.build(bindings), subjects(bindings), binding))
		for _, build := range body {
			effects = append(effects, build(bindings))
		}
		return mage.Sequence(effects...)
	}, nil
}
