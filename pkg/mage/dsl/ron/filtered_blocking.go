package ron

import "github.com/benprew/mage-go/pkg/mage"

func compilePermanentBlockingRule(n Node, ctx scope) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 {
		return compileCombinedAuraRestrictions(n, operations, ctx)
	}
	operation := operations[0]
	switch operation.Name {
	case "CantBeEnchantedByAnEnchantment":
		return compileAuraRestriction(n, operation, ctx)
	case "CantBeTheTargetOfSpells":
		return compileSpellTargetRestriction(n, operation, ctx)
	case "CantAttack":
		if _, err := symbol(operation); err != nil {
			return nil, err
		}
		subject, err := compilePermanentRef(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		if subject.kind != permanentSource && subject.kind != permanentHost {
			return nil, unsupported(n.Args[0], "static attack reference")
		}
		return func() mage.CardOption {
			return mage.WithAbility(mage.Static(mage.CantAttack(subject.build(instanceBindings{}))))
		}, nil
	case "CantAttackUnlessDefendingPlayer":
		return compileDefendingLandRule(n, operation, ctx)
	case "CanAttackAsThoughItDidntHaveDefender", "CanAttackAsThoughItHadHaste":
		return compileStaticAttackPermission(n, operation, ctx)
	case "MustAttack", "CanBlockAnAdditionalCreature":
		return compileSourceCombatAttribute(n, operation, ctx)
	case "CantBeBlockedExceptByDefenders", "CantBeBlockedByDefenders", "CantBlockAttackers":
	default:
		return compileStaticUntapRule(n, ctx)
	}
	subject, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if subject.kind != permanentSource && subject.kind != permanentHost {
		return nil, unsupported(n.Args[0], "static blocking reference")
	}
	if operation.Name == "CantBlockAttackers" && subject.kind != permanentSource {
		return nil, unsupported(n.Args[0], "static blocker reference")
	}
	if err := arity(operation, 1); err != nil {
		return nil, err
	}
	filter, err := compileFilter(operation.Args[0])
	if err != nil {
		return nil, err
	}
	if operation.Name == "CantBeBlockedByDefenders" || operation.Name == "CantBlockAttackers" {
		filter = mage.Not(filter)
	}
	return func() mage.CardOption {
		if operation.Name == "CantBlockAttackers" {
			return mage.WithStaticAbility(mage.SourceCanBlockOnly(filter))
		}
		condition := mage.StateConditionFunc("static blocking rule", []mage.StateDependency{mage.StateBattlefield}, func(mage.StateReader, mage.StateContext) bool { return true })
		return mage.WithAbility(mage.Static(mage.While(condition, mage.CantBeBlockedExceptBy(subject.build(instanceBindings{}), filter))))
	}, nil
}
