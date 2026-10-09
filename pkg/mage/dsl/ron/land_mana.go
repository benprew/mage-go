package ron

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type landManaEventPlan struct{ name string }

type landManaEventBindings struct {
	player *mage.PlayerBinding
	types  *mage.ManaTypesBinding
}

func compileLandManaTrigger(event, body Node, ctx scope) (optionFactory, error) {
	attached := event.Name == "WhenAPermanentIsTappedForMana"
	if attached {
		if err := arity(event, 1); err != nil {
			return nil, err
		}
		subject := event.Args[0]
		if subject.Name != "SinglePermanent" {
			return nil, unsupported(subject, "land mana event subject")
		}
		if err := arity(subject, 1); err != nil {
			return nil, err
		}
		ref, err := compilePermanentRef(subject.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		if ref.kind != permanentHost {
			return nil, unsupported(subject.Args[0], "land mana event reference")
		}
	} else {
		if err := arity(event, 2); err != nil {
			return nil, err
		}
		if event.Args[0].Kind != Symbol || event.Args[0].Name != "AnyPlayer" {
			return nil, unsupported(event.Args[0], "land mana event player filter")
		}
		filter := event.Args[1]
		if filter.Name != "IsCardtype" {
			return nil, unsupported(filter, "land mana event filter")
		}
		if err := arity(filter, 1); err != nil {
			return nil, err
		}
		if filter.Args[0].Kind != Symbol || filter.Args[0].Name != "Land" {
			return nil, unsupported(filter.Args[0], "land mana event card type")
		}
	}
	ctx.landManaEvent = &landManaEventPlan{name: "Trigger_AnyManaTypeProduced"}
	ctx.permanentEvent = &permanentEventPlan{name: "Trigger_ThatPermanent"}
	plan, err := compileActions(body, ctx)
	if err != nil {
		return nil, err
	}
	// The payment solver must use the same bonus as mana production.
	if bonus, ok := compileLandManaBonus(body, attached); ok {
		return bonus, nil
	}
	if usesManaAction(body) {
		return nil, unsupported(body, "land mana bonus action shape")
	}
	if attached {
		return nil, unsupported(body, "attached mana event action shape")
	}
	return func() mage.CardOption {
		land := mage.BindPermanent(ctx.permanentEvent.name)
		player := mage.BindPlayer("Trigger_ThatPlayer")
		types := mage.BindManaTypes(ctx.landManaEvent.name)
		actions := plan.buildWithBindings(instanceBindings{
			tapped:   map[*permanentEventPlan]*mage.PermanentBinding{ctx.permanentEvent: land},
			landMana: map[*landManaEventPlan]landManaEventBindings{ctx.landManaEvent: {player: player, types: types}},
		})
		pattern := mage.LandTappedForMana().WithLand(land).WithPlayer(player).WithManaTypes(types)
		trigger := mage.WhenEvent(pattern, mage.Sequence(actions.effects...)).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger.WithManaTapTextRevision(), actions))
	}, nil
}

func compileLandManaBonus(body Node, attached bool) (optionFactory, bool) {
	if body.Name != "ActionList" || len(body.Args) != 1 || body.Args[0].Kind != List || len(body.Args[0].Args) != 1 {
		return nil, false
	}
	action := body.Args[0].Args[0]
	if action.Name != "PlayerAction" || len(action.Args) != 2 {
		return nil, false
	}
	player, mana := action.Args[0], action.Args[1]
	if mana.Name != "AddMana" || len(mana.Args) != 1 {
		return nil, false
	}
	production := mana.Args[0]
	if !attached && player.Kind == Symbol && player.Name == "Trigger_ThatPlayer" && production.Kind == Symbol && production.Name == "Trigger_AnyManaTypeProduced" {
		return func() mage.CardOption {
			return mage.WithAbility(mage.NewManaFlareAbility(mage.IsLand).WithManaTapTextRevision())
		}, true
	}
	if attached && player.Name == "ControllerOfPermanent" && len(player.Args) == 1 && player.Args[0].Kind == Symbol && player.Args[0].Name == "Trigger_ThatPermanent" {
		productions, err := compileMana(production)
		if err == nil && len(productions) == 1 && productions[0].Color < core.AnyColor {
			return func() mage.CardOption {
				return mage.WithAbility(mage.NewAttachedManaBonusAbility(productions[0].Color).WithManaTapTextRevision())
			}, true
		}
	}
	return nil, false
}

func usesManaAction(n Node) bool {
	if n.Name == "AddMana" || n.Name == "AddManaRepeated" {
		return true
	}
	return slices.ContainsFunc(n.Args, usesManaAction)
}

func compilePlayerMana(n Node, ctx scope) ([]effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	if ctx.landManaEvent == nil {
		return nil, unsupported(n, "mana action outside a land mana event player scope")
	}
	production := n.Args[0]
	if production.Name == "Trigger_AnyManaTypeProduced" {
		if production.Kind != Symbol {
			return nil, problem(production, "unbound or incompatible mana types reference")
		}
		return []effectFactory{func(bindings instanceBindings) mage.Effect {
			return mage.AddManaOfProducedTypeToPlayer(ctx.player.build(bindings), bindings.landMana[ctx.landManaEvent].types, mage.Fixed(1))
		}}, nil
	}
	productions, err := compileMana(production)
	if err != nil {
		return nil, err
	}
	var effects []effectFactory
	for _, p := range productions {
		if p.Color >= core.AnyColor {
			return nil, unsupported(production, "explicit-player mana production")
		}
		effects = append(effects, func(bindings instanceBindings) mage.Effect {
			return mage.AddManaToPlayer(ctx.player.build(bindings), p.Color, mage.Fixed(p.Amount))
		})
	}
	return effects, nil
}
