package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileManaLandSubtype(n Node) (mage.PermanentFilter, error) {
	if n.Name != "IsLandType" {
		return mage.PermanentFilter{}, unsupported(n, "mana bonus land filter")
	}
	if err := arity(n, 1); err != nil {
		return mage.PermanentFilter{}, err
	}
	name, err := symbol(n.Args[0])
	if err != nil {
		return mage.PermanentFilter{}, err
	}
	subtype := subtypeName(name)
	// The event must use a known land subtype.
	pattern := mage.OpponentLandBecomesTapped().OfSubtype(subtype)
	declaration := mage.NewEnchantment("Mana subtype declaration", "", mage.WithAbility(mage.WhenEvent(pattern, mage.GainLifeFor(mage.You, mage.Fixed(0)))))
	if err := mage.ValidateCardDefinition(declaration.Name(), declaration); err != nil {
		return mage.PermanentFilter{}, problem(n.Args[0], err.Error())
	}
	return mage.And(mage.IsLand, mage.HasSubType(subtype)), nil
}

func compileFixedBonusMana(n Node) (core.Color, error) {
	if n.Name != "AddMana" {
		return core.Colorless, unsupported(n, "fixed mana bonus action")
	}
	if err := arity(n, 1); err != nil {
		return core.Colorless, err
	}
	productions, err := compileMana(n.Args[0])
	if err != nil {
		return core.Colorless, err
	}
	if len(productions) != 1 || productions[0].Amount != 1 || productions[0].Color >= core.AnyColor {
		return core.Colorless, unsupported(n.Args[0], "one concrete bonus mana")
	}
	return productions[0].Color, nil
}

func singleManaBonusAction(body Node) (Node, error) {
	if body.Name != "ActionList" {
		return Node{}, unsupported(body, "mana bonus body")
	}
	if err := arity(body, 1); err != nil {
		return Node{}, err
	}
	parts, err := list(body.Args[0])
	if err != nil {
		return Node{}, err
	}
	if len(parts) != 1 {
		return Node{}, problem(body, "supply exactly one mana bonus action")
	}
	return parts[0], nil
}

func compileSubtypeManaTrigger(event, body Node) (optionFactory, error) {
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	filter, err := compileManaLandSubtype(event.Args[0])
	if err != nil {
		return nil, err
	}
	action, err := singleManaBonusAction(body)
	if err != nil {
		return nil, err
	}
	if action.Name != "PlayerAction" {
		return nil, unsupported(action, "mana bonus player action")
	}
	if err := arity(action, 2); err != nil {
		return nil, err
	}
	player := action.Args[0]
	if player.Name != "ControllerOfPermanent" {
		return nil, unsupported(player, "mana bonus player")
	}
	if err := arity(player, 1); err != nil {
		return nil, err
	}
	if player.Args[0].Kind != Symbol || player.Args[0].Name != "Trigger_ThatPermanent" {
		return nil, problem(player.Args[0], "mana bonus requires the event permanent controller")
	}
	color, err := compileFixedBonusMana(action.Args[1])
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.NewControllerManaBonusAbility(filter, color).WithManaTapTextRevision())
	}, nil
}

func compileSubtypeManaBonusUntil(n Node) (effectFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	if n.Args[2].Kind != Symbol || n.Args[2].Name != "UntilEndOfTurn" {
		return nil, unsupported(n.Args[2], "mana bonus duration")
	}
	event := n.Args[0]
	if err := arity(event, 2); err != nil {
		return nil, err
	}
	if event.Args[0].Kind != Symbol || event.Args[0].Name != "AnyPlayer" {
		return nil, unsupported(event.Args[0], "mana bonus player filter")
	}
	filter, err := compileManaLandSubtype(event.Args[1])
	if err != nil {
		return nil, err
	}
	action, err := singleManaBonusAction(n.Args[1])
	if err != nil {
		return nil, err
	}
	color, err := compileFixedBonusMana(action)
	if err != nil {
		return nil, err
	}
	return func(instanceBindings) mage.Effect { return mage.LandManaBonusThisTurn(filter, color) }, nil
}
