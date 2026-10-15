package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileCreatureEnchantRemoval(n Node, subject permanentReferencePlan) (func(instanceBindings) mage.ContinuousModification, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	enchant := n.Args[0]
	if enchant.Name != "Enchant" || enchant.Kind != Tuple {
		return nil, unsupported(enchant, "enchant removal")
	}
	if err := arity(enchant, 1); err != nil {
		return nil, err
	}
	filter := enchant.Args[0]
	if filter.Name != "IsCardtype" || filter.Kind != Tuple {
		return nil, unsupported(filter, "fixed creature enchant removal")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	if filter.Args[0].Kind != Symbol || filter.Args[0].Name != "Creature" {
		return nil, unsupported(filter.Args[0], "fixed creature enchant type")
	}
	return func(bindings instanceBindings) mage.ContinuousModification {
		return mage.RemoveCreatureEnchant(subject.build(bindings))
	}, nil
}
