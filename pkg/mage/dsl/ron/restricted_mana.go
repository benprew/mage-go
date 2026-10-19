package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileRestrictedRepeatedMana(n Node, ctx scope) ([]effectFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	if n.Args[0].Kind != Symbol || n.Args[0].Name != "AnyManaColor" {
		return nil, unsupported(n.Args[0], "restricted one-color mana production")
	}
	restriction, err := compileManaSpendingRestriction(n.Args[2])
	if err != nil {
		return nil, err
	}
	value, err := compileInstanceValue(n.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	return []effectFactory{func(bindings instanceBindings) mage.Effect {
		return (mage.ManaProductionInstruction{Subject: mage.You, Color: core.AnyColor, Amount: value(bindings), Restriction: restriction}).Effect()
	}}, nil
}

func compileManaSpendingRestriction(n Node) (core.ManaRestriction, error) {
	if n.Name != "CanOnlySpendOnSpells" {
		return nil, unsupported(n, "mana spending restriction")
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	filter := n.Args[0]
	if filter.Name != "IsCardtype" {
		return nil, unsupported(filter, "mana spending spell filter")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	if filter.Args[0].Kind != Symbol {
		return nil, unsupported(filter.Args[0], "mana spending spell type")
	}
	switch filter.Args[0].Name {
	case "Creature":
		return core.CreatureSpellsOnly{}, nil
	case "Artifact":
		return core.ArtifactSpellsOnly{}, nil
	default:
		return nil, unsupported(filter.Args[0], "mana spending spell type")
	}
}

func compileRestrictedFixedMana(n Node) ([]effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	restriction, err := compileManaSpendingRestriction(n.Args[1])
	if err != nil {
		return nil, err
	}
	productions, err := compileFixedRestrictedProduction(n.Args[0])
	if err != nil {
		return nil, err
	}
	return []effectFactory{func(instanceBindings) mage.Effect {
		return mage.AddRestrictedMana(productions, restriction)
	}}, nil
}

func compileFixedRestrictedProduction(n Node) ([]mage.ManaProduction, error) {
	if n.Name == "And" {
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return nil, err
		}
		if len(parts) == 0 {
			return nil, problem(n, "supply nonempty fixed mana production")
		}
		var productions []mage.ManaProduction
		for _, part := range parts {
			child, err := compileFixedRestrictedProduction(part)
			if err != nil {
				return nil, err
			}
			productions = append(productions, child...)
		}
		return productions, nil
	}
	productions, err := compileMana(n)
	if err != nil {
		return nil, err
	}
	if productions[0].Color == core.AnyColor {
		return nil, unsupported(n, "fixed restricted mana production")
	}
	return productions, nil
}
