package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileCombatPrevention(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	event := n.Args[0]
	if event.Name == "CombatDamageWouldBeDealtByACreature" {
		return compileCombatCollectionPrevention(n, ctx)
	}
	if event.Name == "DamageWouldBeDealtToRecipient" || event.Name == "DamageWouldBeDealtByAPermanentToRecipient" {
		return compileRecipientPrevention(n, ctx)
	}
	var source *permanentReferencePlan
	var toAndBy *permanentReferencePlan
	if event.Name == "CombatDamageWouldBeDealtByCreature" {
		if err := arity(event, 1); err != nil {
			return nil, err
		}
		ref, err := compilePermanentRef(event.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		source = &ref
	} else if event.Name == "Or" {
		ref, err := compileCombatPreventionUnion(event, ctx)
		if err != nil {
			return nil, err
		}
		toAndBy = &ref
	} else if event.Kind != Symbol || event.Name != "CombatDamageWouldBeDealt" {
		return nil, unsupported(event, "combat prevention event")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 {
		return nil, problem(n.Args[1], "combat prevention requires exactly one operation")
	}
	operation := operations[0]
	if operation.Kind != Symbol || operation.Name != "PreventThatDamage" {
		return nil, unsupported(operation, "combat prevention operation")
	}
	duration := n.Args[2]
	if duration.Kind != Symbol || duration.Name != "UntilEndOfTurn" {
		return nil, unsupported(duration, "combat prevention duration")
	}
	if toAndBy != nil {
		return func(bindings instanceBindings) mage.Effect {
			return mage.PreventCombatDamageToAndBy(toAndBy.build(bindings))
		}, nil
	}
	if source != nil {
		return func(bindings instanceBindings) mage.Effect {
			return mage.PreventCombatDamageByThisTurn(source.build(bindings))
		}, nil
	}
	return func(instanceBindings) mage.Effect { return mage.PreventAllCombatDamage() }, nil
}

func compileCombatPreventionUnion(n Node, ctx scope) (permanentReferencePlan, error) {
	if err := arity(n, 1); err != nil {
		return permanentReferencePlan{}, err
	}
	events, err := list(n.Args[0])
	if err != nil {
		return permanentReferencePlan{}, err
	}
	if len(events) != 2 {
		return permanentReferencePlan{}, problem(n, "combat prevention requires one recipient and one source event")
	}
	var recipient, source *permanentReferencePlan
	for _, event := range events {
		if err := arity(event, 1); err != nil {
			return permanentReferencePlan{}, err
		}
		switch event.Name {
		case "CombatDamageWouldBeDealtToRecipient":
			if recipient != nil {
				return permanentReferencePlan{}, problem(event, "duplicate combat prevention recipient")
			}
			wrapper := event.Args[0]
			if wrapper.Name != "Permanent" {
				return permanentReferencePlan{}, unsupported(wrapper, "combat prevention recipient")
			}
			if err := arity(wrapper, 1); err != nil {
				return permanentReferencePlan{}, err
			}
			ref, err := compilePermanentRef(wrapper.Args[0], ctx)
			if err != nil {
				return permanentReferencePlan{}, err
			}
			recipient = &ref
		case "CombatDamageWouldBeDealtByCreature":
			if source != nil {
				return permanentReferencePlan{}, problem(event, "duplicate combat prevention source")
			}
			ref, err := compilePermanentRef(event.Args[0], ctx)
			if err != nil {
				return permanentReferencePlan{}, err
			}
			source = &ref
		default:
			return permanentReferencePlan{}, unsupported(event, "combat prevention union event")
		}
	}
	if recipient == nil || source == nil || *recipient != *source {
		return permanentReferencePlan{}, problem(n, "combat prevention recipient and source must use the same permanent reference")
	}
	return *source, nil
}
