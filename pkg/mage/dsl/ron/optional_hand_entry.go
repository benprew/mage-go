package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileOptionalHandEntry(n Node) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	filter, err := compileEntryHandFilter(n.Args[0])
	if err != nil {
		return nil, err
	}
	entries, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(entries) != 1 || entries[0].Kind != Symbol || entries[0].Name != "EntersNormally" {
		return nil, unsupported(n.Args[1], "hand entry operations")
	}
	return func(instanceBindings) mage.Effect { return mage.MayPutFromHandOntoBattlefield(mage.You, filter) }, nil
}

func compileEntryHandFilter(n Node) (mage.CardFilter, error) {
	if n.Kind == Symbol && n.Name == "IsPermanent" {
		return mage.IsPermanentCard, nil
	}
	if err := arity(n, 1); err != nil {
		return mage.CardFilter{}, err
	}
	switch n.Name {
	case "And", "Or":
		parts, err := list(n.Args[0])
		if err != nil {
			return mage.CardFilter{}, err
		}
		if len(parts) == 0 {
			return mage.CardFilter{}, problem(n, "hand entry filter requires operands")
		}
		filters := make([]mage.CardFilter, len(parts))
		for i, part := range parts {
			filters[i], err = compileEntryHandFilter(part)
			if err != nil {
				return mage.CardFilter{}, err
			}
		}
		return mage.NewCardFilter(n.Name, func(card mage.Card) bool {
			for _, filter := range filters {
				matches := filter.Match(card)
				if n.Name == "And" && !matches {
					return false
				}
				if n.Name == "Or" && matches {
					return true
				}
			}
			return n.Name == "And"
		}), nil
	case "IsSupertype":
		name, err := symbol(n.Args[0])
		if err != nil {
			return mage.CardFilter{}, err
		}
		kind, ok := map[string]core.SuperType{"Basic": core.SuperBasic, "Legendary": core.SuperLegendary, "Snow": core.SuperSnow, "World": core.SuperWorld}[name]
		if !ok {
			return mage.CardFilter{}, unsupported(n.Args[0], "hand entry supertype")
		}
		return mage.NewCardFilter(n.Name, func(card mage.Card) bool { return card.HasSuperType(kind) }), nil
	case "IsLandType", "IsCreatureType":
		name, err := symbol(n.Args[0])
		if err != nil {
			return mage.CardFilter{}, err
		}
		name = subtypeName(name)
		family := mage.SubtypeCreature
		if n.Name == "IsLandType" {
			family = mage.SubtypeLand
		}
		if err := mage.ValidateSubtypeFamily(family, name); err != nil {
			return mage.CardFilter{}, problem(n.Args[0], err.Error())
		}
		return mage.HasSubTypeCardFilter(name), nil
	default:
		return compileHandCardFilter(n)
	}
}
