package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileLastTurnAttackRule(n, operation Node, ctx scope) (optionFactory, error) {
	if !ctx.permanent || ctx.resolving || ctx.drawReplacementBody || ctx.enteringSource {
		return nil, problem(n, "last-turn attack restriction requires a static permanent context")
	}
	if err := arity(operation, 1); err != nil {
		return nil, err
	}
	exception := operation.Args[0]
	if exception.Name != "ExceptFor" {
		return nil, unsupported(exception, "player attack exception")
	}
	if err := arity(exception, 1); err != nil {
		return nil, err
	}
	condition, err := compileLastTurnActivity(exception.Args[0])
	if err != nil {
		return nil, err
	}
	subjects, err := compileStaticPermanentFilter(n.Args[0])
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.Static(mage.CantAttackEachIf(subjects.build(), mage.NotCondition(condition))))
	}, nil
}

func compileLastTurnActivity(n Node) (mage.StateCondition, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	if n.Name == "Or" {
		parts, err := list(n.Args[0])
		if err != nil {
			return nil, err
		}
		if len(parts) == 0 {
			return nil, problem(n, "last-turn activity requires operands")
		}
		conditions := make([]mage.StateCondition, len(parts))
		for i, part := range parts {
			conditions[i], err = compileLastTurnActivity(part)
			if err != nil {
				return nil, err
			}
		}
		return mage.AnyCondition(conditions...), nil
	}
	operand, err := symbol(n.Args[0])
	if err != nil {
		return nil, err
	}
	switch n.Name {
	case "CastASpellDuringTheirLastTurn":
		if operand != "AnySpell" {
			return nil, unsupported(n.Args[0], "last-turn cast filter")
		}
		return mage.CastSpellDuringLastTurn(mage.CombatDefendingPlayer()), nil
	case "PutAPermanentOnBattleDuringTheirLastTurn":
		if operand != "IsNonToken" {
			return nil, unsupported(n.Args[0], "last-turn entry filter")
		}
		return mage.PutNontokenPermanentDuringLastTurn(mage.CombatDefendingPlayer()), nil
	default:
		return nil, unsupported(n, "last-turn player activity")
	}
}
