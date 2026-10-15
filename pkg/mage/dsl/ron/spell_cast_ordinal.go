package ron

func compileInstantOrdinalTrigger(trigger, body Node, ctx scope) (optionFactory, error) {
	if err := arity(trigger, 3); err != nil {
		return nil, err
	}
	player, comparison, filter := trigger.Args[0], trigger.Args[1], trigger.Args[2]
	if player.Kind != Symbol || player.Name != "Opponent" {
		return nil, unsupported(player, "instant ordinal player filter")
	}
	if comparison.Name != "NotEqualTo" {
		return nil, unsupported(comparison, "instant ordinal comparison")
	}
	if err := arity(comparison, 1); err != nil {
		return nil, err
	}
	value := comparison.Args[0]
	if value.Name != "Integer" {
		return nil, unsupported(value, "instant ordinal value")
	}
	if err := arity(value, 1); err != nil {
		return nil, err
	}
	if value.Args[0].Kind != IntegerNode || value.Args[0].Integer != 1 {
		return nil, unsupported(value, "instant ordinal value")
	}
	if filter.Name != "IsCardtype" {
		return nil, unsupported(filter, "instant ordinal spell filter")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	if filter.Args[0].Kind != Symbol || filter.Args[0].Name != "Instant" {
		return nil, unsupported(filter.Args[0], "instant ordinal spell type")
	}
	event := trigger
	event.Name = "WhenAPlayerCastsASpell"
	event.Args = []Node{player, filter}
	return compileSpellCastTrigger(event, body, ctx, true)
}
