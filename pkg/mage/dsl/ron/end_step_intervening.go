package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileEndStepIntervening(n Node, ctx scope) (optionFactory, error) {
	condition, err := compileEndStepCondition(n.Args[1])
	if err != nil {
		return nil, err
	}
	return compileBoundStepTriggerWithCondition(n.Args[0], n.Args[2], ctx, mage.EachEndStep, mage.YourEndStep, "end-step", condition)
}

func compileEndStepCondition(n Node) (mage.StateCondition, error) {
	switch n.Name {
	case "NoPermanentsPassFilter":
		if err := arity(n, 2); err != nil {
			return nil, err
		}
		zone := n.Args[1]
		if zone.Kind != Symbol || zone.Name != "OnTheBattlefield" {
			return nil, unsupported(zone, "end-step condition zone")
		}
		filter, err := compileFilter(n.Args[0])
		if err != nil {
			return nil, err
		}
		return mage.BattlefieldHasNone(filter), nil
	case "PermanentPassesFilter":
		if err := arity(n, 2); err != nil {
			return nil, err
		}
		source := n.Args[0]
		if source.Kind != Symbol || source.Name != "ThisPermanent" {
			return nil, unsupported(source, "end-step condition source")
		}
		filter := n.Args[1]
		if filter.Name != "DealtDamageToAnyPlayerThisTurn" {
			return nil, unsupported(filter, "end-step history filter")
		}
		if err := arity(filter, 1); err != nil {
			return nil, err
		}
		player := filter.Args[0]
		if player.Kind != Symbol || player.Name != "Opponent" {
			return nil, unsupported(player, "end-step damage player")
		}
		return mage.SourceDamagedOpponentThisTurnCondition(), nil
	default:
		return nil, unsupported(n, "end-step intervening-if condition")
	}
}
