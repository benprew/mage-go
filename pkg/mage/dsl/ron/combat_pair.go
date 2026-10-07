package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileCombatPairTrigger(event, body Node, ctx scope) (optionFactory, error) {
	union := event.Name == "Or"
	if union {
		if err := arity(event, 1); err != nil {
			return nil, err
		}
		parts, err := list(event.Args[0])
		if err != nil {
			return nil, err
		}
		if len(parts) != 2 {
			return nil, problem(event, "combat pair union requires both directions")
		}
		first, second := parts[0], parts[1]
		if first.Name == "WhenACreatureBecomesBlockedByACreature" {
			first, second = second, first
		}
		if first.Name != "WhenACreatureBlocksACreature" || second.Name != "WhenACreatureBecomesBlockedByACreature" {
			return nil, unsupported(event, "combat pair union")
		}
		if err := arity(first, 2); err != nil {
			return nil, err
		}
		if err := arity(second, 2); err != nil {
			return nil, err
		}
		if definitionKey(first.Args[0]) != definitionKey(second.Args[0]) || definitionKey(first.Args[1]) != definitionKey(second.Args[1]) {
			return nil, problem(event, "combat pair union requires the same subject and filter")
		}
		event = first
	}
	if err := arity(event, 2); err != nil {
		return nil, err
	}
	subject := event.Args[0]
	if subject.Name != "SinglePermanent" {
		return nil, unsupported(subject, "combat pair subject")
	}
	if err := arity(subject, 1); err != nil {
		return nil, err
	}
	ref, err := compilePermanentRef(subject.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if ref.kind != permanentSource && ref.kind != permanentHost {
		return nil, unsupported(subject.Args[0], "combat pair source")
	}
	filter, err := compileCombatPairFilter(event.Args[1])
	if err != nil {
		return nil, err
	}
	name := "Trigger_ThatOtherCreature"
	if !union && event.Name == "WhenACreatureBecomesBlockedByACreature" {
		name = "Trigger_TheBlockingCreature"
	}
	ctx.combatOpponent = &permanentEventPlan{name: name}
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		opponent := mage.BindPermanent(name)
		actions := plan.buildWithBindings(instanceBindings{tapped: map[*permanentEventPlan]*mage.PermanentBinding{ctx.combatOpponent: opponent}})
		var pattern mage.EventPattern
		subject := ref.build(instanceBindings{})
		if union {
			pattern = mage.BlocksOrBecomesBlockedByCreatureMatching(subject, filter).WithOpponent(opponent)
		} else if event.Name == "WhenACreatureBlocksACreature" {
			pattern = mage.BlocksCreatureMatching(subject, filter).WithOpponent(opponent)
		} else {
			pattern = mage.BecomesBlockedByCreatureMatching(subject, filter).WithOpponent(opponent)
		}
		trigger := mage.WhenEvent(pattern, mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}

func compileCombatPairFilter(n Node) (mage.PermanentFilter, error) {
	parts := []Node{n}
	if n.Name == "And" {
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, err
		}
		var err error
		parts, err = list(n.Args[0])
		if err != nil {
			return mage.PermanentFilter{}, err
		}
	}
	creature := false
	included := ""
	excluded := ""
	colors := false
	toughness := false
	var filters []mage.PermanentFilter
	for _, part := range parts {
		if err := arity(part, 1); err != nil {
			return mage.PermanentFilter{}, err
		}
		switch part.Name {
		case "IsCardtype":
			if creature || part.Args[0].Kind != Symbol || part.Args[0].Name != "Creature" {
				return mage.PermanentFilter{}, unsupported(part, "combat pair creature filter")
			}
			creature = true
			filters = append(filters, mage.IsCreature)
		case "IsCreatureType":
			if included != "" {
				return mage.PermanentFilter{}, problem(part, "supply one required creature subtype")
			}
			sub, err := symbol(part.Args[0])
			if err != nil {
				return mage.PermanentFilter{}, err
			}
			included = subtypeName(sub)
			if err := mage.ValidateSubtypeFamily(mage.SubtypeCreature, included); err != nil {
				return mage.PermanentFilter{}, problem(part, err.Error())
			}
			filters = append(filters, mage.IsCreature, mage.HasSubType(included))
		case "IsNonCreatureType":
			if excluded != "" {
				return mage.PermanentFilter{}, problem(part, "supply one excluded creature subtype")
			}
			sub, err := symbol(part.Args[0])
			if err != nil {
				return mage.PermanentFilter{}, err
			}
			excluded = subtypeName(sub)
			if err := mage.ValidateSubtypeFamily(mage.SubtypeCreature, excluded); err != nil {
				return mage.PermanentFilter{}, problem(part, err.Error())
			}
			filters = append(filters, mage.Not(mage.HasSubType(excluded)))
		case "ToughnessIs":
			if toughness {
				return mage.PermanentFilter{}, problem(part, "supply one combat pair toughness comparison")
			}
			toughness = true
			filter, err := compileFixedToughnessFilter(part)
			if err != nil {
				return mage.PermanentFilter{}, err
			}
			filters = append(filters, filter)
		case "IsColor", "Or":
			if colors {
				return mage.PermanentFilter{}, problem(part, "supply one combat pair color filter")
			}
			colors = true
			if part.Name == "Or" {
				values, err := list(part.Args[0])
				if err != nil {
					return mage.PermanentFilter{}, err
				}
				if len(values) == 0 {
					return mage.PermanentFilter{}, problem(part, "supply at least one color filter")
				}
				for _, value := range values {
					if value.Name != "IsColor" {
						return mage.PermanentFilter{}, unsupported(value, "combat pair color filter")
					}
				}
			}
			filter, err := compileFilter(part)
			if err != nil {
				return mage.PermanentFilter{}, err
			}
			filters = append(filters, filter)
		default:
			return mage.PermanentFilter{}, unsupported(part, "combat pair filter")
		}
	}
	if !creature && included == "" {
		return mage.PermanentFilter{}, problem(n, "combat pair filter requires the creature type")
	}
	return mage.And(filters...), nil
}
