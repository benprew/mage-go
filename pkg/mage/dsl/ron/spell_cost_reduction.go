package ron

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compilePlayerCostReduction(n Node) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	player, err := symbol(n.Args[0])
	if err != nil {
		return nil, err
	}
	if player != "You" {
		return nil, unsupported(n.Args[0], "spell cost reduction player")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 {
		return nil, problem(n.Args[1], "supply one spell cost reduction operation")
	}
	operation := operations[0]
	if operation.Name == "DamageThatWouldReduceLifeTotalToLessThanNumberReducesItToThatNumberInstead" {
		return compileDamageLifeFloor(operation)
	}
	if operation.Name == "IncreaseSpellCost" {
		return compileStaticSpellCostIncrease(n)
	}
	if operation.Name != "DecreaseSpellCost" {
		return nil, unsupported(operation, "player rule operation")
	}
	if err := arity(operation, 2); err != nil {
		return nil, err
	}
	filter, err := compileCostSpellFilter(operation.Args[0])
	if err != nil {
		return nil, err
	}
	reductions, err := list(operation.Args[1])
	if err != nil {
		return nil, err
	}
	if len(reductions) != 1 {
		return nil, problem(operation.Args[1], "supply one generic spell cost reduction")
	}
	reduction := reductions[0]
	if reduction.Name != "CostReduceGeneric" {
		return nil, unsupported(reduction, "spell cost reduction")
	}
	if err := arity(reduction, 1); err != nil {
		return nil, err
	}
	amount := reduction.Args[0]
	if amount.Kind != IntegerNode || amount.Integer < 0 {
		return nil, problem(amount, "spell cost reduction requires a nonnegative integer literal")
	}
	return func() mage.CardOption {
		return mage.WithStaticAbility(mage.ReduceSpellCostStatic(filter, mage.FixedAmount(amount.Integer), nil))
	}, nil
}

func compileCostSpellFilter(n Node) (mage.SpellPredicate, error) {
	if n.Name == "AnySpell" {
		if _, err := symbol(n); err != nil {
			return nil, err
		}
		return mage.SpellAny(), nil
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	switch n.Name {
	case "IsColor":
		value, err := color(n.Args[0])
		if err != nil {
			return nil, err
		}
		if value == core.Colorless {
			return nil, problem(n.Args[0], "spell color requires one of the five Magic colors")
		}
		filter := mage.HasColorCardFilter(value)
		return func(_ *mage.Game, _ uuid.UUID, card mage.Card, _ uuid.UUID) bool {
			return filter.Match(card)
		}, nil
	case "IsCardtype":
		kind, err := cardType(n.Args[0])
		if err != nil {
			return nil, err
		}
		if kind == core.TypeLand {
			return nil, unsupported(n, "spell cost type filter")
		}
		return mage.SpellHasType(kind), nil
	case "And", "Or":
		parts, err := list(n.Args[0])
		if err != nil {
			return nil, err
		}
		if len(parts) == 0 {
			return nil, problem(n, "supply at least one spell cost filter")
		}
		filters := make([]mage.SpellPredicate, len(parts))
		for i, part := range parts {
			filters[i], err = compileCostSpellFilter(part)
			if err != nil {
				return nil, err
			}
		}
		if n.Name == "And" {
			return mage.SpellsAnd(filters...), nil
		}
		return mage.SpellsOr(filters...), nil
	default:
		return nil, unsupported(n, "spell cost filter")
	}
}
