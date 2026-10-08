package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileEntryCopy(n Node) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	filter, err := compileFilter(n.Args[0])
	if err != nil {
		return nil, err
	}
	modifications, err := compileCopyModifications(n.Args[1])
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.MayCopyAsEntersWithModifications(filter, modifications...))
	}, nil
}

func compileCopyModifications(n Node) ([]mage.CopyModification, error) {
	if n.Kind == Symbol && n.Name == "NoCopyEffects" {
		return nil, nil
	}
	if n.Name != "CopyEffects" {
		return nil, unsupported(n, "entry copy modifications")
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	parts, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(n, "copy modifications require at least one operation")
	}
	modifications := make([]mage.CopyModification, 0, len(parts))
	for _, part := range parts {
		if part.Name != "AddCardtypes" {
			return nil, unsupported(part, "entry copy modifications")
		}
		if err := arity(part, 1); err != nil {
			return nil, err
		}
		types, err := list(part.Args[0])
		if err != nil {
			return nil, err
		}
		if len(types) == 0 {
			return nil, problem(part, "copy type addition requires at least one permanent type")
		}
		kinds := make([]core.CardType, 0, len(types))
		for _, typ := range types {
			kind, err := cardType(typ)
			if err != nil {
				return nil, err
			}
			if kind == core.TypeInstant || kind == core.TypeSorcery {
				return nil, problem(typ, "copy type addition requires permanent types")
			}
			kinds = append(kinds, kind)
		}
		modifications = append(modifications, mage.CopyAddCardTypes(kinds...))
	}
	return modifications, nil
}
