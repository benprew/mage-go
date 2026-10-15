package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

func compileSingleExileEntry(action, continuation Node, ctx scope) (effectFactory, error) {
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, unsupported(action, "single exile entry context")
	}
	if err := arity(action, 2); err != nil {
		return nil, err
	}
	index, err := groupedExileIndex(action.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if !ctx.groupedExile.singular[index] {
		return nil, problem(action.Args[0], "single exile entry requires a singular group")
	}
	options, err := list(action.Args[1])
	if err != nil {
		return nil, err
	}
	if len(options) == 0 || options[0].Kind != Symbol || options[0].Name != "EntersUnderOwnersControl" {
		return nil, problem(action.Args[1], "single exile entry requires owner control")
	}
	tapped := false
	var counters *counterSnapshotPlan
	for _, option := range options[1:] {
		if option.Kind != Symbol {
			return nil, unsupported(option, "single exile entry option")
		}
		switch option.Name {
		case "EntersTapped":
			if tapped {
				return nil, problem(option, "duplicate tapped entry")
			}
			tapped = true
		case "EntersWithNotedCounters":
			if counters != nil || ctx.counterSnapshot == nil {
				return nil, problem(option, "supply one bound counter snapshot")
			}
			counters = ctx.counterSnapshot
		default:
			return nil, unsupported(option, "single exile entry option")
		}
	}
	count := 2
	switch continuation.Name {
	case "If", "Unless":
	case "IfElse":
		count = 3
	default:
		return nil, problem(continuation, "entry requires an adjacent result continuation")
	}
	if err := arity(continuation, count); err != nil {
		return nil, err
	}
	condition := continuation.Args[0]
	if condition.Kind != Symbol || (condition.Name != "CostWasPaid" && condition.Name != "CostWasntPaid") {
		return nil, unsupported(condition, "entry result condition")
	}
	firstSuccess := (condition.Name == "CostWasPaid") != (continuation.Name == "Unless")
	plan := &returnedPermanentPlan{label: "exile entrant at " + action.Position.String()}
	branches := make([][]effectFactory, count-1)
	for i := range branches {
		parts, err := list(continuation.Args[i+1])
		if err != nil {
			return nil, err
		}
		if len(parts) == 0 {
			return nil, problem(continuation.Args[i+1], "entry branch requires at least one action")
		}
		branchCtx := ctx
		if (i == 0) == firstSuccess {
			branchCtx.returnedPermanent = plan
		}
		branches[i], err = compileActionSequence(parts, branchCtx)
		if err != nil {
			return nil, err
		}
	}
	group := ctx.groupedExile
	return func(bindings instanceBindings) mage.Effect {
		bindings.returnedPermanents = maps.Clone(bindings.returnedPermanents)
		if bindings.returnedPermanents == nil {
			bindings.returnedPermanents = make(map[*returnedPermanentPlan]*mage.PermanentBinding)
		}
		result := mage.BindPermanent(plan.label)
		bindings.returnedPermanents[plan] = result
		var entry []mage.BattlefieldEntryOption
		if tapped {
			entry = append(entry, mage.EnterTapped())
		}
		if counters != nil {
			entry = append(entry, mage.EnterWithSavedCounters(bindings.counterSnapshots[counters]))
		}
		bodies := make([]mage.ResolvingEffect, len(branches))
		for i, branch := range branches {
			effects := make([]mage.Effect, len(branch))
			for j, build := range branch {
				effects[j] = build(bindings)
			}
			bodies[i] = mage.Sequence(effects...)
		}
		var success, failure mage.ResolvingEffect
		success = bodies[0]
		if len(bodies) == 2 {
			failure = bodies[1]
		}
		if !firstSuccess {
			success, failure = failure, success
		}
		return mage.PutExiledCardOntoBattlefieldOrElse(mage.SingleExiledCardFrom(bindings.groupedExiles[group][index]), result, success, failure, entry...)
	}, nil
}
