package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileTargetOnlySubtypeFilter(n Node) (mage.CardFilter, error) {
	if err := arity(n, 1); err != nil {
		return mage.CardFilter{}, err
	}
	predicate := n.Args[0]
	if predicate.Name != "IsCreatureType" {
		return mage.CardFilter{}, unsupported(predicate, "target-only subtype predicate")
	}
	if err := arity(predicate, 1); err != nil {
		return mage.CardFilter{}, err
	}
	subtype, err := symbol(predicate.Args[0])
	if err != nil {
		return mage.CardFilter{}, err
	}
	return mage.CanTargetOnlySubtypeFilter(subtypeName(subtype)), nil
}
