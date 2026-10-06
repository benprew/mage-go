package ron

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileCastingWindow(n Node) (mage.StateCondition, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	restriction := n.Args[0]
	if restriction.Name != "CantBeCastUnless" {
		return nil, unsupported(restriction, "casting restriction")
	}
	if err := arity(restriction, 1); err != nil {
		return nil, err
	}
	return compileCastingCondition(restriction.Args[0])
}

func compileCastingCondition(n Node) (mage.StateCondition, error) {
	return compileCastingConditionInCombat(n, hasDuringCombat(n))
}

func hasDuringCombat(n Node) bool {
	if n.Kind == Symbol && n.Name == "IsDuringCombat" {
		return true
	}
	if n.Name == "And" && n.Kind == Tuple && len(n.Args) == 1 && n.Args[0].Kind == List {
		if slices.ContainsFunc(n.Args[0].Args, hasDuringCombat) {
			return true
		}
	}
	return false
}

func compileCastingConditionInCombat(n Node, during bool) (mage.StateCondition, error) {
	if n.Name == "And" {
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return nil, err
		}
		if len(parts) == 0 {
			return nil, problem(n, "supply at least one casting condition")
		}
		conditions := make([]mage.StateCondition, len(parts))
		for i, part := range parts {
			conditions[i], err = compileCastingConditionInCombat(part, during)
			if err != nil {
				return nil, err
			}
		}
		return mage.AllConditions(conditions...), nil
	}
	if n.Name == "IsAPlayersTurn" || n.Name == "IsPlayersTurn" {
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		player, err := symbol(n.Args[0])
		if err != nil {
			return nil, err
		}
		switch player {
		case "You":
			return mage.IsYourTurn(), nil
		case "Opponent":
			return mage.NotCondition(mage.IsYourTurn()), nil
		default:
			return nil, unsupported(n.Args[0], "casting turn player")
		}
	}
	if _, err := symbol(n); err != nil {
		return nil, err
	}
	switch n.Name {
	case "IsDuringCombat":
		return mage.IsDuringSteps(core.BeginCombat, core.DeclareAttackers, core.DeclareBlockers, core.FirstStrikeDamage, core.CombatDamage, core.EndCombat), nil
	case "IsBeforeAttackersDeclared":
		return mage.BeforeCombatStep(core.DeclareAttackers, during), nil
	case "IsBeforeBlockersDeclared":
		return mage.BeforeCombatStep(core.DeclareBlockers, during), nil
	case "IsBeforeCombatDamageStep":
		return mage.BeforeCombatStep(core.CombatDamage, during), nil
	case "IsDuringUpkeep":
		return mage.IsDuringSteps(core.Upkeep), nil
	case "IsDuringDeclareAttackersStep":
		return mage.IsDuringSteps(core.DeclareAttackers), nil
	case "IsDuringDeclareBlockersStep":
		return mage.IsDuringSteps(core.DeclareBlockers), nil
	case "IsAfterUpkeep":
		return mage.AfterFirstUpkeep(), nil
	default:
		return nil, unsupported(n, "casting window condition")
	}
}
