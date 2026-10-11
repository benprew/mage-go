package ron

import "github.com/benprew/mage-go/pkg/mage"

type damageTargetPlan struct{ name string }

func compileDamageTargetRef(n Node, ctx scope) (*damageTargetPlan, error) {
	if n.Kind == Symbol && ctx.damageGroups[n.Name] != nil {
		return ctx.damageGroups[n.Name], nil
	}
	if n.Kind == Symbol && ctx.damageTarget != nil && ctx.damageTarget.name == n.Name {
		return ctx.damageTarget, nil
	}
	return nil, problem(n, "unbound or incompatible damage recipient reference")
}

func compilePlayerPlaneswalkerTarget(n Node, ctx scope) (targetFactory, string, error) {
	if err := arity(n, 2); err != nil {
		return nil, "", err
	}
	player := n.Args[0]
	if player.Kind != Symbol || (player.Name != "AnyPlayer" && player.Name != "Opponent") {
		return nil, "", unsupported(player, "player-or-planeswalker player filter")
	}
	permanent := n.Args[1]
	if permanent.Name != "IsCardtype" {
		return nil, "", unsupported(permanent, "player-or-planeswalker permanent filter")
	}
	if err := arity(permanent, 1); err != nil {
		return nil, "", err
	}
	if permanent.Args[0].Kind != Symbol || permanent.Args[0].Name != "Planeswalker" {
		return nil, "", unsupported(permanent.Args[0], "player-or-planeswalker permanent type")
	}
	return func(instanceBindings) mage.Target {
		if player.Name == "Opponent" {
			return mage.DeclareOpponentOrPlaneswalkerTarget(ctx.targetNamespace + "Ref_TargetPlayerOrPermanent")
		}
		return mage.DeclarePlayerOrPlaneswalkerTarget(ctx.targetNamespace + "Ref_TargetPlayerOrPermanent")
	}, "Ref_TargetPlayerOrPermanent", nil
}
