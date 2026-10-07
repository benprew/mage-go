package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileEndCombatTrigger(event, body Node, ctx scope, condition mage.StateCondition) (optionFactory, error) {
	if _, err := symbol(event); err != nil {
		return nil, err
	}
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		actions := plan.build()
		var options []mage.TriggerOption
		if condition != nil {
			options = append(options, mage.WithInterveningIf(condition))
		}
		trigger := mage.At(mage.EachEndOfCombat, mage.Sequence(actions.effects...), options...).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}

func compileEndCombatIntervening(n Node, ctx scope) (optionFactory, error) {
	condition := n.Args[1]
	if condition.Name != "ACombatPermanentPassesFilter" {
		return nil, unsupported(condition, "end-combat intervening-if condition")
	}
	if err := arity(condition, 2); err != nil {
		return nil, err
	}
	source := condition.Args[0]
	if source.Kind != Symbol || source.Name != "ThisPermanent" {
		return nil, unsupported(source, "end-combat condition source")
	}
	filter := condition.Args[1]
	if filter.Name != "Or" {
		return nil, unsupported(filter, "end-combat history filter")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	parts, err := list(filter.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) != 2 {
		return nil, problem(filter, "supply attacked and blocked combat history")
	}
	seen := map[string]bool{}
	for _, part := range parts {
		if part.Kind != Symbol || (part.Name != "AttackedThisCombat" && part.Name != "BlockedThisCombat") || seen[part.Name] {
			return nil, unsupported(part, "end-combat history filter")
		}
		seen[part.Name] = true
	}
	return compileEndCombatTrigger(n.Args[0], n.Args[2], ctx, mage.SourceAttackedOrBlockedThisCombatCondition())
}
