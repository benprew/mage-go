package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func conditionalManaBody(n Node) bool {
	return n.Name == "ActionList" && len(n.Args) == 1 && n.Args[0].Kind == List && len(n.Args[0].Args) > 0 && n.Args[0].Args[0].Name == "IfElse"
}

func compileConditionalMana(n Node) (optionFactory, error) {
	if n.Name != "Activated" {
		return nil, unsupported(n, "conditional mana activation")
	}
	cost := n.Args[0]
	if cost.Name != "TapPermanent" {
		return nil, unsupported(cost, "conditional mana cost")
	}
	if err := arity(cost, 1); err != nil {
		return nil, err
	}
	if cost.Args[0].Kind != Symbol || cost.Args[0].Name != "ThisPermanent" {
		return nil, unsupported(cost.Args[0], "conditional mana source")
	}
	actions := n.Args[1].Args[0].Args
	if len(actions) != 1 {
		return nil, problem(n.Args[1], "conditional mana requires one branch action")
	}
	branch := actions[0]
	if err := arity(branch, 3); err != nil {
		return nil, err
	}
	condition, err := compileManaCondition(branch.Args[0])
	if err != nil {
		return nil, err
	}
	whenTrue, err := compileManaBranch(branch.Args[1])
	if err != nil {
		return nil, err
	}
	whenFalse, err := compileManaBranch(branch.Args[2])
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.NewConditionalManaAbility(condition, whenTrue, whenFalse))
	}, nil
}

func compileManaCondition(n Node) (mage.StateCondition, error) {
	if n.Name != "PlayerPassesFilter" {
		return nil, unsupported(n, "conditional mana condition")
	}
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	if n.Args[0].Kind != Symbol || n.Args[0].Name != "You" {
		return nil, unsupported(n.Args[0], "conditional mana player")
	}
	return compileManaControlCondition(n.Args[1])
}

func compileManaControlCondition(n Node) (mage.StateCondition, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	if n.Name == "And" {
		parts, err := list(n.Args[0])
		if err != nil {
			return nil, err
		}
		if len(parts) == 0 {
			return nil, problem(n, "supply at least one control condition")
		}
		conditions := make([]mage.StateCondition, len(parts))
		for i, part := range parts {
			conditions[i], err = compileManaControlCondition(part)
			if err != nil {
				return nil, err
			}
		}
		return mage.AllConditions(conditions...), nil
	}
	if n.Name != "ControlsA" {
		return nil, unsupported(n, "conditional mana control condition")
	}
	filter, err := compileManaControlFilter(n.Args[0])
	if err != nil {
		return nil, err
	}
	return mage.PlayerControls(mage.You, filter), nil
}

func compileManaControlFilter(n Node) (mage.PermanentFilter, error) {
	switch n.Name {
	case "And", "Or", "AnyPermanent", "IsCardtype", "IsNonCardtype", "IsLandType", "IsNonLandType", "IsCreatureType", "IsNonCreatureType", "IsEnchantmentType", "IsColor", "IsNonColor", "IsSupertype", "IsNonSupertype":
	default:
		// Payment queries precede the tap cost, so this filter must not read tap status.
		return mage.PermanentFilter{}, unsupported(n, "conditional mana characteristic filter")
	}
	filter, err := compileStaticConditionFilter(n)
	if err != nil {
		return mage.PermanentFilter{}, err
	}
	switch n.Name {
	case "And", "Or":
		for _, part := range n.Args[0].Args {
			if _, err := compileManaControlFilter(part); err != nil {
				return mage.PermanentFilter{}, err
			}
		}
	case "IsLandType", "IsNonLandType", "IsCreatureType", "IsNonCreatureType", "IsEnchantmentType":
		family := mage.SubtypeLand
		switch n.Name {
		case "IsCreatureType", "IsNonCreatureType":
			family = mage.SubtypeCreature
		case "IsEnchantmentType":
			family = mage.SubtypeEnchantment
		}
		if err := mage.ValidateSubtypeFamily(family, subtypeName(n.Args[0].Name)); err != nil {
			return mage.PermanentFilter{}, problem(n, err.Error())
		}
	}
	return filter, nil
}

func compileManaBranch(n Node) ([]mage.ManaProduction, error) {
	actions, err := list(n)
	if err != nil {
		return nil, err
	}
	if len(actions) != 1 || actions[0].Name != "AddMana" {
		return nil, problem(n, "supply one AddMana action per branch")
	}
	action := actions[0]
	if err := arity(action, 1); err != nil {
		return nil, err
	}
	productions, err := compileMana(action.Args[0])
	if err != nil {
		return nil, err
	}
	if len(productions) == 0 {
		return nil, problem(action, "supply fixed mana production")
	}
	for _, production := range productions {
		if production.Color < core.Colorless || production.Color >= core.AnyColor {
			return nil, unsupported(action, "conditional fixed mana production")
		}
	}
	return productions, nil
}
