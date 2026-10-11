package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type graveyardEventPlan struct{ ownerName, controllerName string }

func compileGraveyardTrigger(event, body Node, ctx scope) (optionFactory, error) {
	return compileGraveyardTriggerWithOptions(event, body, ctx)
}

func compileGraveyardTriggerWithOptions(event, body Node, ctx scope, options ...mage.TriggerOption) (optionFactory, error) {
	if err := arity(event, 2); err != nil {
		return nil, err
	}
	if event.Args[1].Kind != Symbol || event.Args[1].Name != "AnyPlayer" {
		return nil, unsupported(event.Args[1], "graveyard destination player filter")
	}
	cardType, controlled, err := compileGraveyardEventFilter(event.Args[0])
	if err != nil {
		return nil, err
	}
	ctx.graveyardEvent = &graveyardEventPlan{ownerName: "Trigger_ThatPlayer", controllerName: "ControllerOfDeadPermanent"}
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		owner := mage.BindPlayer(ctx.graveyardEvent.ownerName)
		controller := mage.BindPlayer(ctx.graveyardEvent.controllerName)
		bindings := instanceBindings{
			graveyardOwners:      map[*graveyardEventPlan]*mage.PlayerBinding{ctx.graveyardEvent: owner},
			graveyardControllers: map[*graveyardEventPlan]*mage.PlayerBinding{ctx.graveyardEvent: controller},
		}
		actions := plan.buildWithBindings(bindings)
		pattern := mage.PermanentDies().OfType(cardType).WithOwner(owner).WithController(controller)
		if controlled {
			pattern = pattern.YouControlled()
		}
		trigger := mage.WhenEvent(pattern, mage.Sequence(actions.effects...), options...).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}

func compileGraveyardEventFilter(n Node) (core.CardType, bool, error) {
	parts := []Node{n}
	if n.Name == "And" {
		if err := arity(n, 1); err != nil {
			return 0, false, err
		}
		var err error
		parts, err = list(n.Args[0])
		if err != nil {
			return 0, false, err
		}
		if len(parts) != 2 {
			return 0, false, problem(n, "graveyard filter requires one type and one controller restriction")
		}
	}
	var cardType core.CardType
	var controlled bool
	for _, part := range parts {
		if err := arity(part, 1); err != nil {
			return 0, false, err
		}
		switch part.Name {
		case "IsCardtype":
			if cardType != 0 {
				return 0, false, problem(part, "duplicate graveyard card type")
			}
			name, err := symbol(part.Args[0])
			if err != nil {
				return 0, false, err
			}
			switch name {
			case "Land":
				cardType = core.TypeLand
			case "Artifact":
				cardType = core.TypeArtifact
			default:
				return 0, false, unsupported(part.Args[0], "graveyard event card type")
			}
		case "ControlledByAPlayer":
			if controlled {
				return 0, false, problem(part, "duplicate graveyard controller restriction")
			}
			player := part.Args[0]
			if player.Name != "SinglePlayer" {
				return 0, false, unsupported(player, "graveyard controller filter")
			}
			if err := arity(player, 1); err != nil {
				return 0, false, err
			}
			if player.Args[0].Kind != Symbol || player.Args[0].Name != "You" {
				return 0, false, unsupported(player.Args[0], "graveyard controller reference")
			}
			controlled = true
		default:
			return 0, false, unsupported(part, "graveyard event permanent filter")
		}
	}
	if cardType == 0 || (n.Name == "And" && !controlled) {
		return 0, false, problem(n, "graveyard filter requires one type and optional controller restriction")
	}
	return cardType, controlled, nil
}
