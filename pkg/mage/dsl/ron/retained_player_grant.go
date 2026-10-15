package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileLastingUpkeepGrant(trigger Node, ctx scope, subject permanentReferencePlan) (effectFactory, error) {
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
	players := event.Args[0]
	if players.Name != "SinglePlayer" {
		return nil, unsupported(players, "granted upkeep player filter")
	}
	if err := arity(players, 1); err != nil {
		return nil, err
	}
	playerNode := players.Args[0]
	if playerNode.Kind == Symbol && playerNode.Name == "You" {
		build, err := compileRecipientUpkeepAbility(trigger)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.Effect {
			return mage.Until(core.Indefinite, mage.TriggeredAbilityModification(subject.build(bindings), build()))
		}, nil
	}
	player, err := compilePlayerRef(playerNode, ctx)
	if err != nil {
		return nil, err
	}
	event.Args = []Node{{Kind: Symbol, Name: "AnyPlayer", Position: players.Position}}
	build, err := compileBoundStepAbility(event, trigger.Args[1], scope{permanent: true}, mage.EachUpkeep, mage.YourUpkeep, "upkeep", nil)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.Until(core.Indefinite, mage.TriggeredAbilityForPlayerModification(subject.build(bindings), player.build(bindings), build()))
	}, nil
}
