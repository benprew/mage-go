package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileCombatPairTrigger(event, body Node, ctx scope) (optionFactory, error) {
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
	if ref.kind != permanentSource {
		return nil, unsupported(subject.Args[0], "combat pair source")
	}
	excluded, err := compileCombatPairFilter(event.Args[1])
	if err != nil {
		return nil, err
	}
	name := "Trigger_ThatOtherCreature"
	if event.Name == "WhenACreatureBecomesBlockedByACreature" {
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
		if event.Name == "WhenACreatureBlocksACreature" {
			p := mage.BlocksCreature(mage.This).WithOpponent(opponent)
			if excluded != "" {
				p = p.WithoutSubtype(excluded)
			}
			pattern = p
		} else {
			p := mage.BecomesBlockedBy(mage.This).WithBlocker(opponent)
			if excluded != "" {
				p = p.WithoutSubtype(excluded)
			}
			pattern = p
		}
		trigger := mage.WhenEvent(pattern, mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}

func compileCombatPairFilter(n Node) (string, error) {
	parts := []Node{n}
	if n.Name == "And" {
		if err := arity(n, 1); err != nil {
			return "", err
		}
		var err error
		parts, err = list(n.Args[0])
		if err != nil {
			return "", err
		}
	}
	creature := false
	excluded := ""
	for _, part := range parts {
		if err := arity(part, 1); err != nil {
			return "", err
		}
		switch part.Name {
		case "IsCardtype":
			if creature || part.Args[0].Kind != Symbol || part.Args[0].Name != "Creature" {
				return "", unsupported(part, "combat pair creature filter")
			}
			creature = true
		case "IsNonCreatureType":
			if excluded != "" {
				return "", problem(part, "supply one excluded creature subtype")
			}
			sub, err := symbol(part.Args[0])
			if err != nil {
				return "", err
			}
			excluded = subtypeName(sub)
			if err := mage.ValidateSubtypeFamily(mage.SubtypeCreature, excluded); err != nil {
				return "", problem(part, err.Error())
			}
		default:
			return "", unsupported(part, "combat pair filter")
		}
	}
	if !creature {
		return "", problem(n, "combat pair filter requires the creature type")
	}
	return excluded, nil
}
