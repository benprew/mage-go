package ron

import "github.com/benprew/mage-go/pkg/mage"

const forcedManaAction = "ActivateAManaAbilityOfEachPermanentAndLoseUnspentMana"

func compileForcedManaPlayer(n Node, ctx scope) (playerReferencePlan, error) {
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return playerReferencePlan{}, problem(n, "forced mana activation requires an ordinary resolving body")
	}
	if n.Name != "PlayerAction" {
		return playerReferencePlan{}, problem(n, "forced mana activation requires an explicit player action")
	}
	if err := arity(n, 2); err != nil {
		return playerReferencePlan{}, err
	}
	player, err := compilePlayerRef(n.Args[0], ctx)
	if err != nil {
		return playerReferencePlan{}, err
	}
	action := n.Args[1]
	if err := arity(action, 1); err != nil {
		return playerReferencePlan{}, err
	}
	filter := action.Args[0]
	if filter.Name != "And" {
		return playerReferencePlan{}, unsupported(filter, "forced mana land filter")
	}
	if err := arity(filter, 1); err != nil {
		return playerReferencePlan{}, err
	}
	parts, err := list(filter.Args[0])
	if err != nil {
		return playerReferencePlan{}, err
	}
	if len(parts) != 2 {
		return playerReferencePlan{}, problem(filter, "supply a land type and ActionPlayer controller")
	}
	land, controller := parts[0], parts[1]
	if controller.Name == "IsCardtype" {
		land, controller = controller, land
	}
	if land.Name != "IsCardtype" || len(land.Args) != 1 || land.Args[0].Kind != Symbol || land.Args[0].Name != "Land" {
		return playerReferencePlan{}, unsupported(land, "forced mana land type")
	}
	if controller.Name != "ControlledByAPlayer" || len(controller.Args) != 1 {
		return playerReferencePlan{}, unsupported(controller, "forced mana controller")
	}
	single := controller.Args[0]
	if single.Name != "SinglePlayer" || len(single.Args) != 1 || single.Args[0].Kind != Symbol || single.Args[0].Name != "ActionPlayer" {
		return playerReferencePlan{}, unsupported(single, "forced mana action player")
	}
	return player, nil
}

func compileForcedMana(n Node, ctx scope) ([]effectFactory, error) {
	player, err := compileForcedManaPlayer(n, ctx)
	if err != nil {
		return nil, err
	}
	return []effectFactory{func(bindings instanceBindings) mage.Effect {
		saved := mage.BindPlayer("forced mana player " + n.Position.String())
		ref := mage.FromPlayer(saved)
		return mage.Sequence(mage.SnapshotPlayer(player.build(bindings), saved), mage.ActivateManaAbilityOfEach(ref, mage.IsLand), mage.LoseUnspentManaFor(ref))
	}}, nil
}
