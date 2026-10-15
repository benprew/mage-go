package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileGraveyardHandContinuation(payment, continuation Node, ctx scope) (effectFactory, error) {
	if !ctx.resolving || ctx.drawReplacementBody || ctx.enteringSource {
		return nil, unsupported(payment, "graveyard move continuation context")
	}
	instruction := payment.Args[0]
	if err := arity(instruction, 1); err != nil {
		return nil, err
	}
	card, err := compileCardRef(instruction.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if card.exile != nil || (card.sourceZone != nil && card.sourceZone.zone != core.ZoneGraveyard) || (card.target != nil && card.target.ante) {
		return nil, problem(instruction.Args[0], "graveyard return requires a graveyard card reference")
	}
	count := 2
	switch continuation.Name {
	case "If", "Unless":
	case "IfElse":
		count = 3
	default:
		return nil, problem(continuation, "zone move requires an adjacent result continuation")
	}
	if err := arity(continuation, count); err != nil {
		return nil, err
	}
	condition := continuation.Args[0]
	if condition.Kind != Symbol || (condition.Name != "CostWasPaid" && condition.Name != "CostWasntPaid") {
		return nil, unsupported(condition, "zone move result condition")
	}
	branches := make([][]effectFactory, count-1)
	for i := range branches {
		parts, err := list(continuation.Args[i+1])
		if err != nil {
			return nil, err
		}
		if len(parts) == 0 {
			return nil, problem(continuation.Args[i+1], "zone move branch requires at least one action")
		}
		branches[i], err = compileActionSequence(parts, ctx)
		if err != nil {
			return nil, err
		}
	}
	return func(bindings instanceBindings) mage.Effect {
		bodies := make([]mage.ResolvingEffect, len(branches))
		for i, branch := range branches {
			effects := make([]mage.Effect, len(branch))
			for j, build := range branch {
				effects[j] = build(bindings)
			}
			bodies[i] = mage.Sequence(effects...)
		}
		var moved, notMoved mage.ResolvingEffect
		moved = bodies[0]
		if len(bodies) == 2 {
			notMoved = bodies[1]
		}
		if (condition.Name == "CostWasntPaid") != (continuation.Name == "Unless") {
			moved, notMoved = notMoved, moved
		}
		return mage.ReturnCardToHandOrElse(card.build(bindings), moved, notMoved)
	}, nil
}
