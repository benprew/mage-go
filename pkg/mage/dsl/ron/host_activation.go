package ron

import "github.com/benprew/mage-go/pkg/mage"

type abilityEventPlan struct{ name string }

func compileHostActivationTrigger(event, body Node, ctx scope) (optionFactory, error) {
	if err := arity(event, 2); err != nil {
		return nil, err
	}
	if event.Args[0].Kind != Symbol || event.Args[0].Name != "AnyPlayer" {
		return nil, unsupported(event.Args[0], "host activation player filter")
	}
	filter := event.Args[1]
	if filter.Name != "And" {
		return nil, unsupported(filter, "host activation filter")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	parts, err := list(filter.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) != 3 {
		return nil, problem(filter, "supply one host, tap cost and nonmana filter")
	}
	seen := make(map[string]bool)
	for _, part := range parts {
		if seen[part.Name] {
			return nil, problem(part, "duplicate host activation filter")
		}
		seen[part.Name] = true
		switch part.Name {
		case "AbilityOfPermanent":
			if err := arity(part, 1); err != nil {
				return nil, err
			}
			ref, err := compilePermanentRef(part.Args[0], ctx)
			if err != nil {
				return nil, err
			}
			if ref.kind != permanentHost {
				return nil, unsupported(part.Args[0], "host activation source")
			}
		case "HasTapSelfInCost", "NonManaAbility":
			if _, err := symbol(part); err != nil {
				return nil, err
			}
		default:
			return nil, unsupported(part, "host activation filter")
		}
	}
	ctx.abilityEvent = &abilityEventPlan{name: "Trigger_ThatActivatedAbility"}
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		binding := mage.BindAbility(ctx.abilityEvent.name)
		actions := plan.buildWithBindings(instanceBindings{eventAbilities: map[*abilityEventPlan]*mage.AbilityBinding{ctx.abilityEvent: binding}})
		pattern := mage.ActivatedAbilityEvent().FromAttachedPermanent().WithTapCost().NonMana().WithAbility(binding)
		trigger := mage.WhenEvent(pattern, mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}
