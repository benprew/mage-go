package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileAttackRestrictionGrant(n Node, ctx scope) (permanentReferencePlan, error) {
	var empty permanentReferencePlan
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return empty, unsupported(n, "static ability payment context")
	}
	if err := arity(n, 3); err != nil {
		return empty, err
	}
	subject, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return empty, err
	}
	if subject.kind != permanentSource {
		return empty, unsupported(n.Args[0], "static ability payment subject")
	}
	if n.Args[2].Kind != Symbol || n.Args[2].Name != "UntilEndOfCombat" {
		return empty, unsupported(n.Args[2], "static ability payment duration")
	}
	parts, err := list(n.Args[1])
	if err != nil {
		return empty, err
	}
	if len(parts) != 1 || parts[0].Name != "AddAbility" {
		return empty, unsupported(n.Args[1], "static ability payment modification")
	}
	if err := arity(parts[0], 1); err != nil {
		return empty, err
	}
	abilities, err := list(parts[0].Args[0])
	if err != nil {
		return empty, err
	}
	if len(abilities) != 1 || abilities[0].Name != "PermanentRuleEffect" {
		return empty, unsupported(parts[0], "static ability payment ability")
	}
	rule := abilities[0]
	if err := arity(rule, 2); err != nil {
		return empty, err
	}
	if rule.Args[0].Kind != Symbol || rule.Args[0].Name != "ThisPermanent" {
		return empty, unsupported(rule.Args[0], "granted attack restriction subject")
	}
	operations, err := list(rule.Args[1])
	if err != nil {
		return empty, err
	}
	if len(operations) != 1 || operations[0].Kind != Symbol || operations[0].Name != "CantAttack" {
		return empty, unsupported(rule.Args[1], "granted attack restriction")
	}
	return subject, nil
}

func compileAttackTapping(n Node, ctx scope) (effectFactory, error) {
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, unsupported(n, "attack tapping context")
	}
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	player, err := compilePlayerRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if n.Args[2].Kind != Symbol || n.Args[2].Name != "UntilEndOfCombat" {
		return nil, unsupported(n.Args[2], "attack tapping duration")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) == 0 {
		return nil, problem(n.Args[1], "supply an attack tapping operation")
	}
	var effects []effectFactory
	for _, op := range operations {
		if op.Name != "AttackingDoesntCauseCreaturesToTapIf" {
			return nil, unsupported(op, "attack tapping operation")
		}
		if err := arity(op, 2); err != nil {
			return nil, err
		}
		filter, err := compileFilter(op.Args[0])
		if err != nil {
			return nil, err
		}
		condition := op.Args[1]
		if condition.Name != "PermanentPassesFilter" {
			return nil, unsupported(condition, "attack tapping condition")
		}
		if err := arity(condition, 2); err != nil {
			return nil, err
		}
		ref, err := compilePermanentRef(condition.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		if ref.kind != permanentSource || condition.Args[1].Kind != Symbol || condition.Args[1].Name != "IsUntapped" {
			return nil, unsupported(condition, "attack tapping source condition")
		}
		effects = append(effects, func(bindings instanceBindings) mage.Effect {
			return mage.AttacksWithoutTappingUntil(player.build(bindings), filter, mage.PermanentIsUntapped(mage.This), core.EndOfCombat)
		})
	}
	return func(bindings instanceBindings) mage.Effect {
		built := make([]mage.Effect, len(effects))
		for i, build := range effects {
			built[i] = build(bindings)
		}
		return mage.Sequence(built...)
	}, nil
}
