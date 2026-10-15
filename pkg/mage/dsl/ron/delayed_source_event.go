package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileDelayedSourceEvent(n Node, ctx scope) (effectFactory, error) {
	if !ctx.permanent || !ctx.resolving || ctx.drawReplacementBody || ctx.sourceZone != nil {
		return nil, problem(n, "delayed source events require an ordinary resolving permanent source")
	}
	events := []Node{n.Args[0]}
	if events[0].Name == "Or" {
		union := events[0]
		if err := arity(union, 1); err != nil {
			return nil, err
		}
		var err error
		events, err = list(union.Args[0])
		if err != nil {
			return nil, err
		}
		if len(events) != 2 {
			return nil, problem(union, "supply one leave event and one untap event")
		}
	}
	leaves, untaps := false, false
	for _, event := range events {
		if err := arity(event, 1); err != nil {
			return nil, err
		}
		if event.Args[0].Kind != Symbol || event.Args[0].Name != "ThisPermanent" {
			return nil, problem(event.Args[0], "delayed source events require ThisPermanent")
		}
		switch event.Name {
		case "WhenPermanentLeavesTheBattlefield":
			if leaves {
				return nil, problem(event, "duplicate delayed leave event")
			}
			leaves = true
		case "WhenPermanentBecomesUntapped":
			if untaps {
				return nil, problem(event, "duplicate delayed untap event")
			}
			untaps = true
		default:
			return nil, unsupported(event, "delayed source event")
		}
	}
	body := n.Args[1]
	if body.Name != "ActionList" {
		return nil, unsupported(body, "delayed source body")
	}
	if err := arity(body, 1); err != nil {
		return nil, err
	}
	parts, err := list(body.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, problem(body, "delayed source event requires at least one action")
	}
	ctx.delayedStep = true
	effects, err := compileActionSequence(parts, ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		values := make([]mage.Effect, len(effects))
		for i, build := range effects {
			values[i] = build(bindings)
		}
		action := mage.Sequence(values...)
		if leaves && untaps {
			return mage.WhenLeavesOrUntaps(mage.This, action)
		}
		if leaves {
			return mage.WhenLeaves(mage.This, action)
		}
		return mage.WhenUntaps(mage.This, action)
	}, nil
}
