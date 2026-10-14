package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type sourceZonePlan struct {
	zone core.Zone
	name string
}

func compileSourceZoneUpkeep(n Node) (optionFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	rule := n.Args[0]
	if rule.Name != "TriggerA" && rule.Name != "TriggerI" {
		return nil, unsupported(rule, "source-zone trigger")
	}
	count := 2
	if rule.Name == "TriggerI" {
		count = 3
	}
	if err := arity(rule, count); err != nil {
		return nil, err
	}
	event := rule.Args[0]
	if event.Name != "AtTheBeginningOfAPlayersUpkeep" {
		return nil, unsupported(event, "source-zone event")
	}
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	players := event.Args[0]
	if players.Name != "SinglePlayer" {
		return nil, unsupported(players, "source-zone upkeep player")
	}
	if err := arity(players, 1); err != nil {
		return nil, err
	}
	if players.Args[0].Kind != Symbol || players.Args[0].Name != "You" {
		return nil, unsupported(players.Args[0], "source-zone upkeep owner")
	}
	body := rule.Args[count-1]
	if body.Name != "ActionList" {
		return nil, unsupported(body, "source-zone action list")
	}
	source := &sourceZonePlan{zone: core.ZoneGraveyard, name: "ThisGraveyardCard"}
	if n.Name == "FromExile" {
		source = &sourceZonePlan{zone: core.ZoneExile, name: "ThisExiledCard"}
	}
	ctx := scope{sourceZone: source, stepEvent: &stepEventPlan{playerName: "Trigger_ThatPlayer"}}
	var condition func(instanceBindings) mage.StateCondition
	if rule.Name == "TriggerI" {
		var err error
		condition, err = compileSourceZoneCondition(rule.Args[1], ctx)
		if err != nil {
			return nil, err
		}
	}
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		card, player := mage.BindCard(ctx.sourceZone.name), mage.BindPlayer(ctx.stepEvent.playerName)
		bindings := instanceBindings{
			sourceZones: map[*sourceZonePlan]*mage.CardBinding{ctx.sourceZone: card},
			steps:       map[*stepEventPlan]*mage.PlayerBinding{ctx.stepEvent: player},
		}
		actions := plan.buildWithBindings(bindings)
		options := []mage.TriggerOption{mage.WithSourceCardInZone(ctx.sourceZone.zone, card)}
		if condition != nil {
			options = append(options, mage.WithInterveningIf(condition(bindings)))
		}
		trigger := mage.WhenEvent(mage.TurnStepBegins(mage.YourUpkeep, player), mage.Sequence(actions.effects...), options...).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}
