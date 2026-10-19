package ron

import "github.com/benprew/mage-go/pkg/mage"

func compilePlayerTarget(n Node, ctx scope) (targetFactory, string, error) {
	if err := arity(n, 1); err != nil {
		return nil, "", err
	}
	players := n.Args[0]
	if players.Kind != Symbol || (players.Name != "AnyPlayer" && players.Name != "Opponent" && players.Name != "AttackedThisTurn") {
		return nil, "", unsupported(players, "player target filter")
	}
	return func(instanceBindings) mage.Target {
		var options []mage.PlayerTargetOption
		if players.Name == "Opponent" {
			options = append(options, mage.TargetOpponentOnly())
		}
		if players.Name == "AttackedThisTurn" {
			options = append(options, mage.TargetPlayerAttackedThisTurn())
		}
		return mage.DeclarePlayerTarget(ctx.targetNamespace+"Ref_TargetPlayer", options...)
	}, "Ref_TargetPlayer", nil
}
