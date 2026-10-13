package ron

import "github.com/benprew/mage-go/pkg/mage"

type statePermanentSetPlan struct{ name string }

func compileStatePermanentSetTrigger(event, body Node, ctx scope) (optionFactory, error) {
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	subjects, err := compileStatePermanentSubjects(event.Args[0])
	if err != nil {
		return nil, err
	}
	result := &statePermanentSetPlan{name: "state members " + event.Position.String()}
	ctx.statePermanents = result
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		binding := mage.BindPermanentSet(result.name)
		actions := plan.buildWithBindings(instanceBindings{statePermanents: map[*statePermanentSetPlan]*mage.PermanentSetBinding{result: binding}})
		trigger := mage.WhenPermanentsPresent(subjects, binding, mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}

func compileStatePermanentSubjects(n Node) (mage.PermanentSet, error) {
	switch n.Name {
	case "Other":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		if n.Args[0].Kind != Symbol || n.Args[0].Name != "ThisPermanent" {
			return nil, unsupported(n.Args[0], "state source exclusion")
		}
		return mage.WithoutPermanents(mage.MatchingPermanents(mage.AnyPermanent), mage.SingletonPermanent(mage.This)), nil
	case "And", "Or":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return nil, err
		}
		if len(parts) == 0 {
			return nil, problem(n, "state subjects require a nonempty group")
		}
		sets := make([]mage.PermanentSet, len(parts))
		for i, part := range parts {
			sets[i], err = compileStatePermanentSubjects(part)
			if err != nil {
				return nil, err
			}
		}
		if n.Name == "And" {
			return mage.IntersectPermanents(sets...), nil
		}
		return mage.UnionPermanents(sets...), nil
	default:
		filter, err := compileFilter(n)
		if err != nil {
			return nil, err
		}
		return mage.MatchingPermanents(filter), nil
	}
}
