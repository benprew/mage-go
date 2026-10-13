package ron

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileTargetTypeDependency(n Node, ctx scope) (mage.PermanentFilter, permanentTargetOptionsFactory, error) {
	count := 1
	if n.Name == "SharesACardtypeWithPermanentFromAmongCardtypes" {
		count = 2
	}
	if err := arity(n, count); err != nil {
		return mage.PermanentFilter{}, nil, err
	}
	ref := n.Args[0]
	group := ctx.permanentGroups[ref.Name]
	if ref.Kind != Symbol || group == nil {
		return mage.PermanentFilter{}, nil, problem(ref, "target comparison requires an earlier singular permanent target")
	}
	var types []core.CardType
	if count == 2 {
		parts, err := list(n.Args[1])
		if err != nil {
			return mage.PermanentFilter{}, nil, err
		}
		if len(parts) == 0 {
			return mage.PermanentFilter{}, nil, problem(n.Args[1], "target comparison requires a nonempty type list")
		}
		for _, part := range parts {
			kind, err := cardType(part)
			if err != nil {
				return mage.PermanentFilter{}, nil, err
			}
			if slices.Contains(types, kind) {
				return mage.PermanentFilter{}, nil, problem(part, "target comparison requires distinct types")
			}
			types = append(types, kind)
		}
	}
	return mage.AnyPermanent, func(bindings instanceBindings) []mage.PermanentTargetOption {
		return []mage.PermanentTargetOption{mage.TargetSharesCardTypeWith(bindings.permanents[group], types...)}
	}, nil
}
