package ron

import "github.com/benprew/mage-go/pkg/mage"

// TriggerA adds an action list to a triggered ability.
func TriggerA(trigger *mage.GenericTriggered, actions Actions) *mage.GenericTriggered {
	for _, effect := range actions.effects {
		trigger.AddEffect(effect)
	}
	for _, target := range actions.targets {
		trigger.AddTarget(target)
	}
	return trigger
}

func compileTriggerRule(n Node, ctx scope) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	trigger := n.Args[0]
	var buildTrigger func(mage.Effect) *mage.GenericTriggered
	switch trigger.Name {
	case "WhenAPermanentIsPutIntoAPlayersGraveyard":
		return compileGraveyardTrigger(trigger, n.Args[1], ctx)
	case "WhenAPlayerTapsAPermanentForMana", "WhenAPermanentIsTappedForMana":
		return compileLandManaTrigger(trigger, n.Args[1], ctx)
	case "WhenAPermanentBecomesTapped":
		if trigger.Kind == Tuple && len(trigger.Args) == 1 && trigger.Args[0].Name == "SinglePermanent" {
			return compileHostTapTrigger(trigger, n.Args[1], ctx)
		}
		var err error
		buildTrigger, err = compileOpponentLandTap(trigger)
		if err != nil {
			return nil, err
		}
	case "WhenAPermanentEntersTheBattlefield", "WhenAPermanentDies":
		if err := arity(trigger, 1); err != nil {
			return nil, err
		}
		permanents := trigger.Args[0]
		if permanents.Name != "SinglePermanent" {
			if trigger.Name != "WhenAPermanentDies" {
				return nil, unsupported(permanents, "trigger subject")
			}
			var err error
			buildTrigger, err = compileCreatureDeathTrigger(permanents)
			if err != nil {
				return nil, err
			}
			break
		}
		if err := arity(permanents, 1); err != nil {
			return nil, err
		}
		if permanents.Args[0].Kind != Symbol || permanents.Args[0].Name != "ThisPermanent" {
			return nil, unsupported(permanents.Args[0], "trigger reference")
		}
		if trigger.Name == "WhenAPermanentEntersTheBattlefield" {
			buildTrigger = func(effect mage.Effect) *mage.GenericTriggered {
				return mage.WhenEvent(mage.Enters(mage.This), effect).(*mage.GenericTriggered)
			}
		} else {
			buildTrigger = func(effect mage.Effect) *mage.GenericTriggered { return mage.DiesTrigger(effect, false) }
		}
	case "AtTheBeginningOfAPlayersUpkeep":
		return compileBoundStepTrigger(trigger, n.Args[1], ctx, mage.EachUpkeep, mage.YourUpkeep, "upkeep")
	case "AtTheBeginningOfAPlayersEndStep":
		return compileEndStepTrigger(trigger, n.Args[1], ctx)
	case "WhenAPlayerCastsASpell":
		if err := arity(trigger, 2); err != nil {
			return nil, err
		}
		players := trigger.Args[0]
		if players.Kind != Symbol || (players.Name != "AnyPlayer" && players.Name != "Opponent") {
			return nil, unsupported(players, "spell-cast player filter")
		}
		filter, err := compileSpellFilter(trigger.Args[1])
		if err != nil {
			return nil, err
		}
		ctx.castEvent = &castEventPlan{spellName: "Trigger_ThatSpell", casterName: "Trigger_ThatPlayer"}
		plan, err := compileActions(n.Args[1], ctx)
		if err != nil {
			return nil, err
		}
		return func() mage.CardOption {
			bindings := castEventBindings{spell: mage.BindSpell(ctx.castEvent.spellName), caster: mage.BindPlayer(ctx.castEvent.casterName)}
			actions := plan.buildWithEvents(map[*castEventPlan]castEventBindings{ctx.castEvent: bindings})
			pattern := mage.SpellCast().WithSpell(bindings.spell).WithCaster(bindings.caster)
			if players.Name == "Opponent" {
				pattern = pattern.ByOpponent()
			}
			trigger := mage.WhenEvent(pattern, mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
			trigger.AndConditionData(mage.SpellCastMatchesCardFilters{Filters: []mage.CardFilter{filter}})
			actions.effects = nil
			return mage.WithAbility(TriggerA(trigger, actions))
		}, nil
	default:
		return nil, unsupported(trigger, "trigger")
	}
	plan, err := compileActions(n.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		actions := plan.build()
		trigger := buildTrigger(mage.Sequence(actions.effects...))
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}
