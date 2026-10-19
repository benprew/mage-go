package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type retainedBlockerFactory func(instanceBindings) mage.RetainedBlockerFilter

func compileRetainedBlockerEligibility(n, operation Node, subject permanentReferencePlan, ctx scope) (effectFactory, error) {
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody || ctx.permanentLoop == nil || ctx.selectedPile == nil || ctx.selectedPile.loop != ctx.permanentLoop || n.Args[0].Kind != Symbol || n.Args[0].Name != "EachablePermanent" {
		return nil, problem(n, "retained blocking requires a preceding pile choice for this resolving loop member")
	}
	if err := arity(operation, 1); err != nil {
		return nil, err
	}
	if n.Args[2].Kind != Symbol || n.Args[2].Name != "UntilEndOfCombat" {
		return nil, unsupported(n.Args[2], "retained blocking duration")
	}
	filter, err := compileRetainedBlockerFilter(operation.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.RetainBlockerEligibility(core.EndOfCombat, subject.build(bindings), mage.MatchingPermanents(mage.IsCreature), filter(bindings))
	}, nil
}

func compileRetainedBlockerFilter(n Node, ctx scope) (retainedBlockerFactory, error) {
	switch n.Name {
	case "InThePileChosenForPermanent":
		set, err := compileSelectedPileRef(n, ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.RetainedBlockerFilter { return mage.SavedBlockers(set(bindings)) }, nil
	case "And", "Or":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return nil, err
		}
		if len(parts) == 0 {
			return nil, problem(n, "retained blocker filter requires at least one operand")
		}
		plans := make([]retainedBlockerFactory, len(parts))
		for i, part := range parts {
			plans[i], err = compileRetainedBlockerFilter(part, ctx)
			if err != nil {
				return nil, err
			}
		}
		return func(bindings instanceBindings) mage.RetainedBlockerFilter {
			filters := make([]mage.RetainedBlockerFilter, len(plans))
			for i, build := range plans {
				filters[i] = build(bindings)
			}
			if n.Name == "And" {
				return mage.AllRetainedBlockers(filters...)
			}
			return mage.AnyRetainedBlockers(filters...)
		}, nil
	default:
		filter, err := compileFilter(n)
		if err != nil {
			return nil, err
		}
		return func(instanceBindings) mage.RetainedBlockerFilter { return mage.CurrentBlockers(filter) }, nil
	}
}
