package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileHostCombatTrigger(event, body Node, ctx scope) (optionFactory, error) {
	parts := []Node{event}
	union := event.Name == "Or"
	if union {
		if err := arity(event, 1); err != nil {
			return nil, err
		}
		var err error
		parts, err = list(event.Args[0])
		if err != nil {
			return nil, err
		}
		if len(parts) != 2 {
			return nil, problem(event, "host combat union requires attack and block")
		}
		if parts[0].Name == "WhenACreatureBlocks" {
			parts = []Node{parts[1], parts[0]}
		}
		if parts[0].Name != "WhenACreatureAttacks" || parts[1].Name != "WhenACreatureBlocks" {
			return nil, unsupported(event, "host combat union")
		}
	}
	for _, part := range parts {
		if part.Name != "WhenACreatureAttacks" && part.Name != "WhenACreatureBlocks" && part.Name != "WhenACreatureAttacksAndIsntBlocked" {
			return nil, unsupported(part, "host combat event")
		}
		if err := arity(part, 1); err != nil {
			return nil, err
		}
		subject := part.Args[0]
		if subject.Name != "SinglePermanent" {
			return nil, unsupported(subject, "host combat subject")
		}
		if err := arity(subject, 1); err != nil {
			return nil, err
		}
		ref, err := compilePermanentRef(subject.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		if ref.kind != permanentHost {
			return nil, unsupported(subject.Args[0], "host combat reference")
		}
	}
	ctx.hostCombat = &permanentEventPlan{name: "Trigger_ThatCreature"}
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		host := mage.BindPermanent(ctx.hostCombat.name)
		actions := plan.buildWithBindings(instanceBindings{tapped: map[*permanentEventPlan]*mage.PermanentBinding{ctx.hostCombat: host}})
		var pattern mage.EventPattern
		switch {
		case union:
			pattern = mage.AttachedAttacksOrBlocks(host)
		case event.Name == "WhenACreatureAttacks":
			pattern = mage.AttachedAttacks(host)
		case event.Name == "WhenACreatureBlocks":
			pattern = mage.AttachedBlocks(host)
		default:
			pattern = mage.AttachedAttacksUnblocked(host)
		}
		trigger := mage.WhenEvent(pattern, mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}
