package ron

import "github.com/benprew/mage-go/pkg/mage"

type graveyardTargetsPlan struct{ name string }

func compilePlayerGraveyardGroups(n Node, targets []Node, ctx scope) (actionPlan, error) {
	var plan actionPlan
	if !ctx.spellTargets || ctx.permanent || ctx.activation || ctx.delayedStep || ctx.drawReplacementBody {
		return plan, unsupported(n, "player and graveyard target context")
	}
	if len(targets) != 2 || targets[0].Name != "TargetPlayer" || targets[1].Name != "AnyNumberOfTargetGraveyardCards" {
		return plan, unsupported(n, "player and graveyard target groups")
	}
	player, _, err := compileTarget(targets[0], ctx)
	if err != nil {
		return plan, err
	}
	target := targets[1]
	if err := arity(target, 1); err != nil {
		return plan, err
	}
	filter := target.Args[0]
	if filter.Name != "And" {
		return plan, unsupported(filter, "dependent graveyard filter")
	}
	if err := arity(filter, 1); err != nil {
		return plan, err
	}
	parts, err := list(filter.Args[0])
	if err != nil {
		return plan, err
	}
	if len(parts) != 2 {
		return plan, problem(filter, "dependent graveyard filter requires a card type and player")
	}
	var cardFilter mage.CardFilter
	foundType, foundPlayer := false, false
	for _, part := range parts {
		if err := arity(part, 1); err != nil {
			return plan, err
		}
		switch part.Name {
		case "IsCardtype":
			if foundType {
				return plan, problem(part, "duplicate card type")
			}
			foundType = true
			kind, err := cardType(part.Args[0])
			if err != nil {
				return plan, err
			}
			cardFilter = mage.NewCardFilter(kind.String()+" card", func(card mage.Card) bool { return card.HasType(kind) })
		case "InAPlayersGraveyard":
			if foundPlayer {
				return plan, problem(part, "duplicate graveyard player")
			}
			foundPlayer = true
			players := part.Args[0]
			if players.Name != "SinglePlayer" {
				return plan, unsupported(players, "dependent graveyard player")
			}
			if err := arity(players, 1); err != nil {
				return plan, err
			}
			if players.Args[0].Kind != Symbol || players.Args[0].Name != "Ref_TargetPlayer" {
				return plan, problem(players, "graveyard group requires the earlier player target")
			}
		default:
			return plan, unsupported(part, "dependent graveyard filter")
		}
	}
	if !foundType || !foundPlayer {
		return plan, problem(filter, "dependent graveyard filter requires a card type and player")
	}
	ctx.playerTarget = &playerTargetPlan{name: "Ref_TargetPlayer"}
	ctx.graveyardTargets = &graveyardTargetsPlan{name: "Ref_TargetGraveyardCards"}
	plan, err = compileActions(n.Args[1], ctx)
	if err != nil {
		return plan, err
	}
	if len(plan.targets) != 0 {
		return plan, problem(n, "nested target declarations are not supported")
	}
	plan.playerTarget = ctx.playerTarget
	plan.graveyardTargets = ctx.graveyardTargets
	plan.targets = []targetFactory{player, func(bindings instanceBindings) mage.Target {
		return mage.DeclareGraveyardCardTargets(ctx.targetNamespace+ctx.graveyardTargets.name, bindings.players[ctx.playerTarget], cardFilter)
	}}
	return plan, nil
}
func compileGraveyardCardsToTop(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	ref := n.Args[0]
	if ref.Kind != Symbol || !ctx.resolving || !ctx.spellTargets || ctx.permanent || ctx.activation || ctx.enteringSource || ctx.graveyardTargets == nil || ref.Name != ctx.graveyardTargets.name || ctx.delayedStep || ctx.drawReplacementBody {
		return nil, problem(ref, "unbound or incompatible graveyard target collection")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.ReturnGraveyardCardsToLibraryTop(bindings.graveyardTargets[ctx.graveyardTargets], mage.You)
	}, nil
}
