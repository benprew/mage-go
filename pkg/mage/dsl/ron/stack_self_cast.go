package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileStackSelfCast(n Node, ctx scope) (optionFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	rule := n.Args[0]
	if rule.Name != "TriggerA" {
		return nil, unsupported(rule, "stack-source rule")
	}
	if err := arity(rule, 2); err != nil {
		return nil, err
	}
	event := rule.Args[0]
	if event.Name != "WhenAPlayerCastsASpell" {
		return nil, unsupported(event, "stack-source event")
	}
	if err := arity(event, 2); err != nil {
		return nil, err
	}
	player, spell := event.Args[0], event.Args[1]
	if player.Name != "SinglePlayer" {
		return nil, unsupported(player, "self-cast player")
	}
	if err := arity(player, 1); err != nil {
		return nil, err
	}
	if player.Args[0].Kind != Symbol || player.Args[0].Name != "You" {
		return nil, unsupported(player.Args[0], "self-cast player reference")
	}
	if spell.Name != "SingleSpell" {
		return nil, unsupported(spell, "self-cast spell")
	}
	if err := arity(spell, 1); err != nil {
		return nil, err
	}
	if spell.Args[0].Kind != Symbol || spell.Args[0].Name != "ThisSpell" {
		return nil, unsupported(spell.Args[0], "self-cast spell reference")
	}
	ctx.permanent = false
	ctx.aura = false
	ctx.castEvent = &castEventPlan{spellName: "Trigger_ThatSpell", casterName: "Trigger_ThatPlayer"}
	plan, err := compileActions(rule.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		bindings := castEventBindings{spell: mage.BindSpell(ctx.castEvent.spellName), caster: mage.BindPlayer(ctx.castEvent.casterName)}
		actions := plan.buildWithEvents(map[*castEventPlan]castEventBindings{ctx.castEvent: bindings})
		pattern := mage.SourceSpellCast().ByController().WithSpell(bindings.spell).WithCaster(bindings.caster)
		trigger := mage.WhenEvent(pattern, mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}
