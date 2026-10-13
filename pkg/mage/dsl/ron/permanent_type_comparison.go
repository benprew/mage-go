package ron

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compilePermanentTypeComparison(n Node, ctx scope) (permanentSetFactory, error) {
	if ctx.enteringSource || ctx.drawReplacementBody {
		return nil, unsupported(n, "nonresolving card type comparison")
	}
	count := 1
	if n.Name == "SharesACardtypeWithPermanentFromAmongCardtypes" {
		count = 2
	}
	if err := arity(n, count); err != nil {
		return nil, err
	}
	ref, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	var types []core.CardType
	if count == 2 {
		parts, err := list(n.Args[1])
		if err != nil {
			return nil, err
		}
		if len(parts) == 0 {
			return nil, problem(n.Args[1], "card type comparison requires a nonempty type list")
		}
		for _, part := range parts {
			kind, err := cardType(part)
			if err != nil {
				return nil, err
			}
			if slices.Contains(types, kind) {
				return nil, problem(part, "card type comparison requires distinct types")
			}
			types = append(types, kind)
		}
	}
	return func(bindings instanceBindings) mage.PermanentSet {
		return mage.PermanentsSharingCardTypes(ref.build(bindings), mage.MatchingPermanents(mage.AnyPermanent), types...)
	}, nil
}
