package ron

import (
	"maps"

	"github.com/benprew/mage-go/pkg/mage"
)

func compileUnblockedDelay(n Node, ctx scope) (effectFactory, error) {
	event := n.Args[0]
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	subject := event.Args[0]
	if subject.Name != "SinglePermanent" {
		return nil, unsupported(subject, "delayed unblocked subject")
	}
	if err := arity(subject, 1); err != nil {
		return nil, err
	}
	ref, err := compilePermanentRef(subject.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	body := n.Args[1]
	if body.Name != "ActionList" {
		return nil, unsupported(body, "delayed unblocked body")
	}
	if err := arity(body, 1); err != nil {
		return nil, err
	}
	parts, err := list(body.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(body, "delayed unblocked requires at least one action")
	}
	eventPlan := &permanentEventPlan{name: "Trigger_ThatCreature"}
	ctx.unblockedCreature = eventPlan
	ctx.delayedStep = true
	effects, err := compileActionSequence(parts, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		watched := mage.BindPermanent(eventPlan.name)
		child := bindings
		child.tapped = make(map[*permanentEventPlan]*mage.PermanentBinding, len(bindings.tapped)+1)
		maps.Copy(child.tapped, bindings.tapped)
		child.tapped[eventPlan] = watched
		values := make([]mage.Effect, len(effects))
		for i, build := range effects {
			values[i] = build(child)
		}
		return mage.Sequence(mage.Snapshot(ref.build(bindings), watched), mage.WhenAttacksUnblockedThisTurn(mage.From(watched), mage.Sequence(values...)))
	}, nil
}
