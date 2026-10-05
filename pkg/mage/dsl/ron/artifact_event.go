package ron

import "github.com/benprew/mage-go/pkg/mage"

func artifactTypeFilter(n Node) error {
	if err := arity(n, 1); err != nil {
		return err
	}
	if n.Name != "IsCardtype" || n.Args[0].Kind != Symbol || n.Args[0].Name != "Artifact" {
		return unsupported(n, "artifact event filter")
	}
	return nil
}

func compileArtifactTapTrigger(event, body Node, ctx scope) (optionFactory, error) {
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	filter := event.Args[0]
	opponent := false
	if filter.Name == "And" {
		if err := arity(filter, 1); err != nil {
			return nil, err
		}
		parts, err := list(filter.Args[0])
		if err != nil {
			return nil, err
		}
		if len(parts) != 2 {
			return nil, problem(filter, "supply one artifact and one opponent controller filter")
		}
		found := false
		for _, part := range parts {
			if part.Name == "ControlledByAPlayer" {
				if err := arity(part, 1); err != nil {
					return nil, err
				}
				if opponent || part.Args[0].Kind != Symbol || part.Args[0].Name != "Opponent" {
					return nil, unsupported(part, "artifact tap controller filter")
				}
				opponent = true
			} else {
				if found {
					return nil, problem(part, "duplicate artifact filter")
				}
				if err := artifactTypeFilter(part); err != nil {
					return nil, err
				}
				found = true
			}
		}
		if !found || !opponent {
			return nil, problem(filter, "supply one artifact and one opponent controller filter")
		}
	} else if err := artifactTypeFilter(filter); err != nil {
		return nil, err
	}
	return compileArtifactEventBody(body, ctx, func(binding *mage.PermanentBinding, effect mage.Effect) *mage.GenericTriggered {
		pattern := mage.ArtifactBecomesTapped().WithArtifact(binding)
		if opponent {
			pattern = pattern.ByOpponent()
		}
		return mage.WhenEvent(pattern, effect).(*mage.GenericTriggered)
	})
}

func compileArtifactActivationTrigger(event, body Node, ctx scope) (optionFactory, error) {
	if err := arity(event, 2); err != nil {
		return nil, err
	}
	player, filter := event.Args[0], event.Args[1]
	if player.Kind != Symbol || (player.Name != "AnyPlayer" && player.Name != "Opponent") {
		return nil, unsupported(player, "artifact activation player filter")
	}
	if filter.Name != "And" {
		return nil, unsupported(filter, "artifact activation filter")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	parts, err := list(filter.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) != 2 {
		return nil, problem(filter, "supply one artifact source and one non-tap cost filter")
	}
	source, noTap, attached := false, false, false
	for _, part := range parts {
		switch part.Name {
		case "DoesntHaveTapSelfInCost":
			if part.Kind != Symbol || noTap {
				return nil, problem(part, "supply one non-tap cost symbol")
			}
			noTap = true
		case "AbilityOfAPermanent", "AbilityOfPermanent":
			if err := arity(part, 1); err != nil {
				return nil, err
			}
			if source {
				return nil, problem(part, "duplicate activation source filter")
			}
			source = true
			if part.Name == "AbilityOfAPermanent" {
				if err := artifactTypeFilter(part.Args[0]); err != nil {
					return nil, err
				}
			} else {
				ref, err := compilePermanentRef(part.Args[0], ctx)
				if err != nil {
					return nil, err
				}
				if ref.kind != permanentHost {
					return nil, unsupported(part, "attached activation source")
				}
				attached = true
			}
		default:
			return nil, unsupported(part, "artifact activation filter")
		}
	}
	if !source || !noTap {
		return nil, problem(filter, "supply one artifact source and one non-tap cost filter")
	}
	return compileArtifactEventBody(body, ctx, func(binding *mage.PermanentBinding, effect mage.Effect) *mage.GenericTriggered {
		pattern := mage.ArtifactAbilityActivatedWithoutTapCost().WithArtifact(binding)
		if attached {
			pattern = pattern.AttachedToSource()
		}
		trigger := mage.WhenEvent(pattern, effect).(*mage.GenericTriggered)
		if player.Name == "Opponent" {
			trigger.AndConditionData(mage.EventPlayerIsOpponent{})
		}
		return trigger
	})
}

func compileArtifactEventBody(body Node, ctx scope, build func(*mage.PermanentBinding, mage.Effect) *mage.GenericTriggered) (optionFactory, error) {
	ctx.permanentEvent = &permanentEventPlan{name: "Trigger_ThatPermanent"}
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		binding := mage.BindPermanent(ctx.permanentEvent.name)
		actions := plan.buildWithBindings(instanceBindings{tapped: map[*permanentEventPlan]*mage.PermanentBinding{ctx.permanentEvent: binding}})
		trigger := build(binding, mage.Sequence(actions.effects...))
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}

func artifactTapFilter(n Node) bool {
	if n.Name == "And" && len(n.Args) == 1 && n.Args[0].Kind == List {
		for _, part := range n.Args[0].Args {
			if part.Name == "IsCardtype" {
				return true
			}
		}
	}
	return false
}
