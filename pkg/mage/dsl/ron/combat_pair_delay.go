package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

func compileCombatPairDelay(n Node, ctx scope) (effectFactory, error) {
	pair, err := compileCombatPairEvent(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if !pair.union || pair.subject.kind != permanentSource {
		return nil, unsupported(n.Args[0], "delayed source combat pair")
	}
	body := n.Args[1]
	if body.Name != "ActionList" {
		return nil, unsupported(body, "delayed combat pair body")
	}
	if err := arity(body, 1); err != nil {
		return nil, err
	}
	parts, err := list(body.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(body, "delayed combat pair requires at least one action")
	}
	event := &permanentEventPlan{name: pair.name}
	ctx.combatOpponent = event
	ctx.delayedStep = true
	ctx.drawReplacementBody = false
	effects, err := compileActionSequence(parts, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		opponent := mage.BindPermanent(pair.name)
		child := bindings
		child.tapped = maps.Clone(bindings.tapped)
		if child.tapped == nil {
			child.tapped = make(map[*permanentEventPlan]*mage.PermanentBinding)
		}
		child.tapped[event] = opponent
		values := make([]mage.Effect, len(effects))
		for i, build := range effects {
			values[i] = build(child)
		}
		return mage.WhenCombatPairUntilEndOfCombat(pair.subject.build(bindings), pair.filter, opponent, mage.Sequence(values...))
	}, nil
}
