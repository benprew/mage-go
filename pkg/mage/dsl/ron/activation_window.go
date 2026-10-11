package ron

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func activationDuringCombat(n Node) bool {
	if n.Name == "ActivateOnlyIf" && n.Kind == Tuple && len(n.Args) == 1 {
		return hasDuringCombat(n.Args[0])
	}
	if n.Name == "And" && n.Kind == Tuple && len(n.Args) == 1 && n.Args[0].Kind == List {
		if slices.ContainsFunc(n.Args[0].Args, activationDuringCombat) {
			return true
		}
	}
	return false
}

func compileActivationWindow(n Node, during bool) (mage.StateCondition, error) {
	if n.Name == "And" {
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return nil, err
		}
		if len(parts) == 0 {
			return nil, problem(n, "supply at least one activation window condition")
		}
		conditions := make([]mage.StateCondition, len(parts))
		for i, part := range parts {
			conditions[i], err = compileActivationWindow(part, during)
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
			return nil, unsupported(n.Args[0], "activation turn player")
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
	default:
		return nil, unsupported(n, "activation window condition")
	}
}
