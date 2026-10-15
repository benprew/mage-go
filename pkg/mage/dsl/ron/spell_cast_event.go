package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileSpellCastTrigger(trigger, body Node, ctx scope, afterFirstInstant bool) (optionFactory, error) {
	if err := arity(trigger, 2); err != nil {
		return nil, err
	}
	players := trigger.Args[0]
	controllerOnly := false
	if players.Name == "SinglePlayer" {
		if err := arity(players, 1); err != nil {
			return nil, err
		}
		if players.Args[0].Kind != Symbol || players.Args[0].Name != "You" {
			return nil, unsupported(players.Args[0], "spell-cast player reference")
		}
		controllerOnly = true
	} else if players.Kind != Symbol || (players.Name != "AnyPlayer" && players.Name != "Opponent") {
		return nil, unsupported(players, "spell-cast player filter")
	}
	filter, err := compileContextualSpellFilter(trigger.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	ctx.castEvent = &castEventPlan{spellName: "Trigger_ThatSpell", casterName: "Trigger_ThatPlayer"}
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		bindings := castEventBindings{spell: mage.BindSpell(ctx.castEvent.spellName), caster: mage.BindPlayer(ctx.castEvent.casterName)}
		actions := plan.buildWithEvents(map[*castEventPlan]castEventBindings{ctx.castEvent: bindings})
		pattern := mage.SpellCast().WithSpell(bindings.spell).WithCaster(bindings.caster).Matching(filter)
		if players.Name == "Opponent" {
			pattern = pattern.ByOpponent()
		} else if controllerOnly {
			pattern = pattern.ByController()
		}
		if afterFirstInstant {
			pattern = pattern.AfterFirstInstant()
		}
		trigger := mage.WhenEvent(pattern, mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		trigger.WithLayeredSpellCastTextRevision()
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}
