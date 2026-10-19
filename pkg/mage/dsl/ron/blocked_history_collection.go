package ron

import "github.com/benprew/mage-go/pkg/mage"

type blockedHistoryCollectionPlan struct {
	blocker permanentReferencePlan
}

func compileBlockedHistoryCollection(n Node, ctx scope) (permanentSetFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	blocker, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	plan := blockedHistoryCollectionPlan{blocker: blocker}
	return plan.build, nil
}

func (plan blockedHistoryCollectionPlan) build(bindings instanceBindings) mage.PermanentSet {
	return mage.BlockedByThisTurn(plan.blocker.build(bindings))
}
