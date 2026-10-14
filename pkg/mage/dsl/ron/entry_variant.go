package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileEntryVariant(n Node) (optionFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	options, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	if len(options) == 0 {
		return nil, problem(n, "entry variants require at least one option")
	}
	variants := make([][]mage.CopyModification, len(options))
	for i, option := range options {
		parts, err := list(option)
		if err != nil {
			return nil, err
		}
		if len(parts) == 0 {
			return nil, problem(option, "entry variants require nonempty options")
		}
		variants[i] = make([]mage.CopyModification, len(parts))
		for j, part := range parts {
			variants[i][j], err = compileEntryVariantModification(part)
			if err != nil {
				return nil, err
			}
		}
	}
	return func() mage.CardOption { return mage.WithAbility(mage.ChooseCopiableVariantAsEnters(variants...)) }, nil
}

func compileEntryVariantModification(n Node) (mage.CopyModification, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	operand := n.Args[0]
	switch n.Name {
	case "SetCopiablePT":
		if operand.Name != "PT" {
			return nil, unsupported(operand, "copiable P/T")
		}
		if err := arity(operand, 2); err != nil {
			return nil, err
		}
		if operand.Args[0].Kind != IntegerNode || operand.Args[1].Kind != IntegerNode {
			return nil, problem(operand, "copiable P/T requires integer values")
		}
		return mage.CopySetPowerToughness(operand.Args[0].Integer, operand.Args[1].Integer), nil
	case "AddCopiableCardtype":
		kind, err := cardType(operand)
		if err != nil {
			return nil, err
		}
		if kind == core.TypeInstant || kind == core.TypeSorcery {
			return nil, problem(operand, "entry variants require permanent types")
		}
		return mage.CopyAddCardTypes(kind), nil
	case "AddCopiableCreatureType":
		name, err := symbol(operand)
		if err != nil {
			return nil, err
		}
		return mage.CopyAddSubTypes(subtypeName(name)), nil
	case "AddCopiableAbility":
		parts, err := list(operand)
		if err != nil {
			return nil, err
		}
		if len(parts) == 0 {
			return nil, problem(operand, "copiable ability addition requires keywords")
		}
		additions := make([]core.Attr, len(parts))
		for i, part := range parts {
			name, err := symbol(part)
			if err != nil {
				return nil, err
			}
			keyword, ok := keywords[name]
			if !ok {
				return nil, unsupported(part, "copiable keyword")
			}
			additions[i] = keyword
		}
		return mage.CopyAddKeywords(additions...), nil
	default:
		return nil, unsupported(n, "copiable entry variant modification")
	}
}
