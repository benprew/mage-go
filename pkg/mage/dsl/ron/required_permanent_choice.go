package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

func compileRequiredPermanentChoice(cost, continuation Node, player playerReferencePlan, ctx scope) (effectFactory, error) {
	if !ctx.resolving || ctx.drawReplacementBody || ctx.enteringSource {
		return nil, unsupported(cost, "required permanent choice context")
	}
	if err := arity(cost, 1); err != nil {
		return nil, err
	}
	filter := cost.Args[0]
	legalAura := false
	if filter.Name == "And" && len(filter.Args) == 1 && filter.Args[0].Kind == List {
		parts := filter.Args[0].Args
		if len(parts) == 2 && parts[1].Name == "CanBeEnchantedByDeadGuest" {
			if parts[1].Kind != Symbol || ctx.hostDeath == nil || !ctx.aura {
				return nil, unsupported(parts[1], "departed Aura host choice")
			}
			legalAura = true
			ctx.hostDeath.departedAura = true
			filter = parts[0]
		}
	}
	subjects, err := compileContextualCollection(filter, ctx)
	if err != nil {
		return nil, err
	}
	count := 2
	switch continuation.Name {
	case "If", "Unless":
	case "IfElse":
		count = 3
	default:
		return nil, problem(continuation, "choice requires an adjacent result continuation")
	}
	if err := arity(continuation, count); err != nil {
		return nil, err
	}
	condition := continuation.Args[0]
	if condition.Kind != Symbol || (condition.Name != "CostWasPaid" && condition.Name != "CostWasntPaid") {
		return nil, unsupported(condition, "choice result condition")
	}
	firstSuccess := (condition.Name == "CostWasPaid") != (continuation.Name == "Unless")
	plan := &permanentChoicePlan{name: "required chosen permanent " + cost.Position.String()}
	branches := make([][]effectFactory, count-1)
	for i := range branches {
		parts, err := list(continuation.Args[i+1])
		if err != nil {
			return nil, err
		}
		if len(parts) == 0 {
			return nil, problem(continuation.Args[i+1], "choice branch requires at least one action")
		}
		branchCtx := ctx
		if (i == 0) == firstSuccess {
			branchCtx.chosenPermanent = plan
			var preceding []*permanentChoicePlan
			if ctx.permanentChoices != nil {
				preceding = ctx.permanentChoices.choices
			}
			branchCtx.permanentChoices = &permanentChoiceHistoryPlan{choices: append(append([]*permanentChoicePlan(nil), preceding...), plan)}
		}
		branches[i], err = compileActionSequence(parts, branchCtx)
		if err != nil {
			return nil, err
		}
	}
	return func(bindings instanceBindings) mage.Effect {
		result := mage.BindPermanentChoice(plan.name)
		bindings.chosenPermanents = maps.Clone(bindings.chosenPermanents)
		if bindings.chosenPermanents == nil {
			bindings.chosenPermanents = make(map[*permanentChoicePlan]*mage.PermanentChoiceBinding)
		}
		bindings.chosenPermanents[plan] = result
		choices := subjects(bindings)
		if legalAura {
			choices = mage.LegalHostsForAuraCard(mage.FromCard(bindings.hostDeaths[ctx.hostDeath].aura), mage.You, choices)
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
		return mage.ChoosePermanentForPlayerOrElse(player.build(bindings), choices, result, success, failure)
	}, nil
}
