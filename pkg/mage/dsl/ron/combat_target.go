package ron

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
)

func containsCombatTarget(n Node) bool {
	return n.Name == "IsBlockingAttacker" || n.Name == "IsBlockedByDefender" || slices.ContainsFunc(n.Args, containsCombatTarget)
}

func compileCombatTarget(n Node, ctx scope) (mage.PermanentFilter, []mage.PermanentTargetOption, error) {
	if n.Name == "Or" {
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, nil, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return mage.PermanentFilter{}, nil, err
		}
		if len(parts) != 2 {
			return mage.PermanentFilter{}, nil, problem(n, "combat target union requires both directions")
		}
		first, second := parts[0], parts[1]
		if first.Name == "IsBlockedByDefender" {
			first, second = second, first
		}
		if first.Name != "IsBlockingAttacker" || second.Name != "IsBlockedByDefender" {
			return mage.PermanentFilter{}, nil, unsupported(n, "combat target union")
		}
		for _, part := range []Node{first, second} {
			if _, _, err := compileCombatTarget(part, ctx); err != nil {
				return mage.PermanentFilter{}, nil, err
			}
		}
		return mage.AnyPermanent, []mage.PermanentTargetOption{mage.TargetInCombatWithSource()}, nil
	}
	if err := arity(n, 1); err != nil {
		return mage.PermanentFilter{}, nil, err
	}
	if n.Args[0].Kind != Symbol || n.Args[0].Name != "ThisPermanent" || !ctx.permanent {
		return mage.PermanentFilter{}, nil, problem(n.Args[0], "combat target requires ThisPermanent")
	}
	option := mage.TargetBlocksSource()
	if n.Name == "IsBlockedByDefender" {
		option = mage.TargetBlockedBySource()
	}
	return mage.AnyPermanent, []mage.PermanentTargetOption{option}, nil
}
