package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileGrantedUpkeepLayer(n Node, parts []Node, ctx scope) (optionFactory, error) {
	if len(parts) != 1 || parts[0].Name != "AddAbility" {
		return nil, unsupported(n.Args[1], "combined triggered ability grant")
	}
	if err := arity(parts[0], 1); err != nil {
		return nil, err
	}
	abilities, err := list(parts[0].Args[0])
	if err != nil {
		return nil, err
	}
	if len(abilities) != 1 || abilities[0].Name != "TriggerA" {
		return nil, problem(parts[0], "supply one granted upkeep trigger")
	}
	trigger := abilities[0]
	if err := arity(trigger, 2); err != nil {
		return nil, err
	}
	event := trigger.Args[0]
	if event.Name != "AtTheBeginningOfAPlayersUpkeep" {
		return nil, unsupported(event, "granted upkeep event")
	}
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	player := event.Args[0]
	if player.Name != "SinglePlayer" {
		return nil, unsupported(player, "granted upkeep player filter")
	}
	if err := arity(player, 1); err != nil {
		return nil, err
	}
	if player.Args[0].Kind != Symbol || player.Args[0].Name != "You" {
		return nil, unsupported(player.Args[0], "granted upkeep player reference")
	}
	// A granted body cannot read the donor's attachment or bindings.
	build, err := compileBoundStepAbility(event, trigger.Args[1], scope{permanent: true}, mage.EachUpkeep, mage.YourUpkeep, "upkeep", nil)
	if err != nil {
		return nil, err
	}
	if n.Name == "EachPermanentLayerEffect" {
		subjects, err := compileStaticPermanentFilter(n.Args[0])
		if err != nil {
			return nil, err
		}
		return func() mage.CardOption {
			return mage.WithAbility(mage.Static(mage.GrantEachTriggeredAbility(subjects.build(), build())))
		}, nil
	}
	if n.Args[0].Kind != Symbol || n.Args[0].Name != "HostPermanent" || !ctx.aura {
		return nil, unsupported(n.Args[0], "granted upkeep subject")
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.Static(mage.GrantTriggeredAbility(mage.AttachedPermanent, build())))
	}, nil
}
