package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileDynamicManaColors(n Node) (optionFactory, error) {
	if n.Name != "Activated" {
		return nil, unsupported(n, "dynamic mana color activation")
	}
	cost := n.Args[0]
	if cost.Name != "TapPermanent" {
		return nil, unsupported(cost, "dynamic mana color cost")
	}
	if err := arity(cost, 1); err != nil {
		return nil, err
	}
	if cost.Args[0].Kind != Symbol || cost.Args[0].Name != "ThisPermanent" {
		return nil, unsupported(cost.Args[0], "dynamic mana color source")
	}
	body := n.Args[1]
	if body.Name != "ActionList" {
		return nil, unsupported(body, "dynamic mana color body")
	}
	if err := arity(body, 1); err != nil {
		return nil, err
	}
	actions, err := list(body.Args[0])
	if err != nil {
		return nil, err
	}
	if len(actions) != 1 || actions[0].Name != "AddMana" {
		return nil, problem(body, "supply one AddMana action")
	}
	if err := arity(actions[0], 1); err != nil {
		return nil, err
	}
	production := actions[0].Args[0]
	if production.Name != "AnyManaColorAPermanentCouldProduce" {
		return nil, unsupported(production, "dynamic mana color production")
	}
	if err := arity(production, 1); err != nil {
		return nil, err
	}
	filter := production.Args[0]
	if filter.Name != "And" {
		return nil, unsupported(filter, "dynamic mana color filter")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	parts, err := list(filter.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) != 2 {
		return nil, problem(filter, "supply one characteristic filter and one opponent controller filter")
	}
	var characteristic mage.PermanentFilter
	var haveCharacteristic, haveOpponent bool
	for _, part := range parts {
		if part.Name == "ControlledByAPlayer" {
			if err := arity(part, 1); err != nil {
				return nil, err
			}
			if haveOpponent || part.Args[0].Kind != Symbol || part.Args[0].Name != "Opponent" {
				return nil, unsupported(part, "dynamic mana opponent filter")
			}
			haveOpponent = true
		} else {
			if haveCharacteristic {
				return nil, problem(part, "duplicate characteristic filter")
			}
			characteristic, err = compileManaControlFilter(part)
			if err != nil {
				return nil, err
			}
			haveCharacteristic = true
		}
	}
	if !haveCharacteristic || !haveOpponent {
		return nil, problem(filter, "supply one characteristic filter and one opponent controller filter")
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.NewManaAbilityOfColorsOpponentPermanentsCouldProduce(characteristic))
	}, nil
}
