package ron

import "github.com/benprew/mage-go/pkg/mage"

type landEntryEventPlan struct{ playerName string }

func compileLandEntryTrigger(event, body Node, ctx scope) (optionFactory, error) {
	actor := event.Name == "WhenAPlayerPutsAPermanentOnTheBattlefield"
	count := 1
	if actor {
		count = 2
	}
	if err := arity(event, count); err != nil {
		return nil, err
	}
	filter := event.Args[count-1]
	subtype := ""
	if actor {
		if event.Args[0].Kind != Symbol || event.Args[0].Name != "AnyPlayer" {
			return nil, unsupported(event.Args[0], "land entry actor filter")
		}
		if filter.Name != "IsLandType" {
			return nil, unsupported(filter, "land entry subtype filter")
		}
		if err := arity(filter, 1); err != nil {
			return nil, err
		}
		name, err := symbol(filter.Args[0])
		if err != nil {
			return nil, err
		}
		subtype = subtypeName(name)
		switch subtype {
		case "Plains", "Island", "Swamp", "Mountain", "Forest":
		default:
			return nil, unsupported(filter.Args[0], "land entry basic land subtype")
		}
		ctx.landEntryEvent = &landEntryEventPlan{playerName: "Trigger_ThatPlayer"}
	} else {
		if filter.Name != "IsCardtype" {
			return nil, unsupported(filter, "land entry type filter")
		}
		if err := arity(filter, 1); err != nil {
			return nil, err
		}
		if filter.Args[0].Kind != Symbol || filter.Args[0].Name != "Land" {
			return nil, unsupported(filter.Args[0], "land entry card type")
		}
	}
	ctx.permanentEvent = &permanentEventPlan{name: "Trigger_ThatPermanent"}
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		land := mage.BindPermanent(ctx.permanentEvent.name)
		bindings := instanceBindings{tapped: map[*permanentEventPlan]*mage.PermanentBinding{ctx.permanentEvent: land}}
		pattern := mage.LandEnters().WithPermanent(land)
		if actor {
			player := mage.BindPlayer(ctx.landEntryEvent.playerName)
			bindings.entryActors = map[*landEntryEventPlan]*mage.PlayerBinding{ctx.landEntryEvent: player}
			pattern = pattern.OfSubtype(subtype).WithActor(player)
		}
		actions := plan.buildWithBindings(bindings)
		trigger := mage.WhenEvent(pattern, mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}
