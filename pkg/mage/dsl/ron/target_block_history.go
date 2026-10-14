package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileTargetBlockHistory(n Node, ctx scope) (mage.PermanentFilter, permanentTargetOptionsFactory, error) {
	if err := arity(n, 1); err != nil {
		return mage.PermanentFilter{}, nil, err
	}
	ref := n.Args[0]
	group := ctx.permanentGroups[ref.Name]
	if ref.Kind != Symbol || group == nil {
		return mage.PermanentFilter{}, nil, problem(ref, "block history target requires an earlier singular permanent target")
	}
	return mage.AnyPermanent, func(bindings instanceBindings) []mage.PermanentTargetOption {
		return []mage.PermanentTargetOption{mage.TargetBlockedByThisTurn(bindings.permanents[group])}
	}, nil
}
