package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileFixedPowerFilter(n Node) (mage.PermanentFilter, error) {
	if err := arity(n, 1); err != nil {
		return mage.PermanentFilter{}, err
	}
	comparison := n.Args[0]
	if err := arity(comparison, 1); err != nil {
		return mage.PermanentFilter{}, err
	}
	value := comparison.Args[0]
	if value.Name != "Integer" {
		return mage.PermanentFilter{}, unsupported(value, "fixed power limit")
	}
	if err := arity(value, 1); err != nil {
		return mage.PermanentFilter{}, err
	}
	if value.Args[0].Kind != IntegerNode {
		return mage.PermanentFilter{}, problem(value.Args[0], "power limit must be an integer")
	}
	limit := value.Args[0].Integer
	switch comparison.Name {
	case "LessThanOrEqualTo":
		return mage.HasPowerLTE(limit), nil
	case "GreaterThanOrEqualTo":
		return mage.HasPowerGTE(limit), nil
	case "EqualTo":
		return mage.HasPowerEqual(limit), nil
	default:
		return mage.PermanentFilter{}, unsupported(comparison, "power comparison")
	}
}

func compileTemporaryCombatRule(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) == 1 && operations[0].Name == "DoesntUntapDuringControllersUntap" {
		if operations[0].Kind != Symbol {
			return nil, unsupported(operations[0], "untap-step operation")
		}
		duration := n.Args[2]
		if duration.Name != "ForAsLongAsPermanentRemainsTapped" {
			return nil, unsupported(duration, "untap-step duration")
		}
		if err := arity(duration, 1); err != nil {
			return nil, err
		}
		source, err := compilePermanentRef(duration.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		if source.kind != permanentSource || !ctx.resolving || !ctx.permanent || ctx.enteringSource || ctx.drawReplacementBody {
			return nil, unsupported(duration, "untap-step source duration")
		}
		return func(bindings instanceBindings) mage.Effect {
			return mage.PreventUntapWhileSourceTapped(subject.build(bindings))
		}, nil
	}
	if len(operations) == 1 && operations[0].Name == "MustAttack" {
		if _, err := symbol(operations[0]); err != nil {
			return nil, err
		}
		if n.Args[2].Kind != Symbol || n.Args[2].Name != "UntilEndOfTurn" {
			return nil, unsupported(n.Args[2], "attack requirement duration")
		}
		return func(bindings instanceBindings) mage.Effect {
			return mage.MustAttackThisTurn(subject.build(bindings))
		}, nil
	}
	if len(operations) == 1 && operations[0].Name == "AssignsNoCombatDamage" {
		if operations[0].Kind != Symbol {
			return nil, unsupported(operations[0], "combat assignment rule")
		}
		if n.Args[2].Kind != Symbol || n.Args[2].Name != "UntilEndOfTurn" {
			return nil, unsupported(n.Args[2], "duration")
		}
		return func(bindings instanceBindings) mage.Effect {
			return mage.Until(core.EndOfTurn, mage.AssignsNoCombatDamage(subject.build(bindings)))
		}, nil
	}
	if len(operations) == 1 && operations[0].Name == "CantAttack" {
		if operations[0].Kind != Symbol {
			return nil, unsupported(operations[0], "temporary attack restriction")
		}
		duration := n.Args[2]
		if duration.Name != "DuringPlayersNextTurn" {
			return nil, unsupported(duration, "attack restriction duration")
		}
		if err := arity(duration, 1); err != nil {
			return nil, err
		}
		player, err := compilePlayerRef(duration.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.Effect {
			return mage.CantAttackDuringNextTurn(subject.build(bindings), player.build(bindings))
		}, nil
	}
	if subject.kind != permanentTarget && subject.kind != permanentDamageTarget && subject.kind != permanentSource {
		return nil, unsupported(n.Args[0], "temporary combat rule reference")
	}
	for _, operation := range operations {
		switch operation.Name {
		case "CantBeRegenerated", "CanBlockAnyNumberOfCreatures", "MustBlockEachAttacker":
			return compileTemporaryRuleList(n, subject, operations)
		}
	}
	if len(operations) != 1 {
		return nil, unsupported(n.Args[1], "temporary blocking operations")
	}
	operation := operations[0]
	var modification func(mage.PermanentRef) mage.ContinuousModification
	switch operation.Name {
	case "CantBlock":
		if operation.Kind != Symbol {
			return nil, unsupported(operation, "temporary blocking operation")
		}
		modification = mage.CantBlock
	case "CantBeBlocked":
		if operation.Kind != Symbol || (subject.kind != permanentTarget && subject.kind != permanentDamageTarget) {
			return nil, unsupported(operation, "temporary blocking operation")
		}
		modification = mage.CantBeBlocked
	case "CanAttackAsThoughItDidntHaveDefender", "CanAttackAsThoughItHadHaste":
		if operation.Kind != Symbol {
			return nil, unsupported(operation, "temporary attack permission")
		}
		modification = mage.AttackAsThoughHaste
		if operation.Name == "CanAttackAsThoughItDidntHaveDefender" {
			modification = func(ref mage.PermanentRef) mage.ContinuousModification {
				return mage.GrantKeywordModification(ref, core.AttrCanAttack)
			}
		}
	case "CantBeBlockedByDefenders":
		if err := arity(operation, 1); err != nil {
			return nil, err
		}
		filter, err := compileFilter(operation.Args[0])
		if err != nil {
			return nil, err
		}
		modification = func(ref mage.PermanentRef) mage.ContinuousModification {
			return mage.CantBeBlockedExceptBy(ref, mage.Not(filter))
		}
	default:
		return nil, unsupported(operation, "temporary blocking operation")
	}
	if n.Args[2].Kind != Symbol || n.Args[2].Name != "UntilEndOfTurn" {
		return nil, unsupported(n.Args[2], "duration")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.Until(core.EndOfTurn, modification(subject.build(bindings)))
	}, nil
}

func compileTemporaryRuleList(n Node, subject permanentReferencePlan, operations []Node) (effectFactory, error) {
	if n.Args[2].Kind != Symbol || n.Args[2].Name != "UntilEndOfTurn" {
		return nil, unsupported(n.Args[2], "duration")
	}
	modifications := make([]func(mage.PermanentRef) mage.ContinuousModification, len(operations))
	for i, operation := range operations {
		if operation.Kind != Symbol {
			return nil, unsupported(operation, "temporary rule operation")
		}
		switch operation.Name {
		case "CantBeRegenerated":
			modifications[i] = func(ref mage.PermanentRef) mage.ContinuousModification {
				return mage.GrantKeywordModification(ref, core.CantRegenerate)
			}
		case "CanBlockAnyNumberOfCreatures":
			modifications[i] = mage.BlockAnyNumber
		case "MustBlockEachAttacker":
			modifications[i] = mage.BlockEachAttacker
		default:
			return nil, unsupported(operation, "temporary rule operation")
		}
	}
	return func(bindings instanceBindings) mage.Effect {
		ref := subject.build(bindings)
		values := make([]mage.ContinuousModification, len(modifications))
		for i, build := range modifications {
			values[i] = build(ref)
		}
		return mage.Until(core.EndOfTurn, values...)
	}, nil
}
