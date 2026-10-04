package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileOptionalUntap(n Node) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	source := n.Args[0]
	if source.Kind != Symbol || source.Name != "ThisPermanent" {
		return nil, unsupported(source, "optional untap source")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 || operations[0].Kind != Symbol || operations[0].Name != "MayChooseNotToUntapDuringUntap" {
		return nil, unsupported(n.Args[1], "optional untap operations")
	}
	return func() mage.CardOption {
		condition := mage.StateConditionFunc("optional untap", []mage.StateDependency{mage.StateBattlefield}, func(mage.StateReader, mage.StateContext) bool { return true })
		return mage.WithAbility(mage.Static(mage.While(condition, mage.GrantKeywordModification(mage.This, core.AttrMayNotUntap))))
	}, nil
}

func compileTappedUpkeep(n Node, ctx scope) (optionFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	event := n.Args[0]
	if event.Name != "AtTheBeginningOfAPlayersUpkeep" {
		return nil, unsupported(event, "intervening-if event")
	}
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	players := event.Args[0]
	if players.Name != "SinglePlayer" {
		return nil, unsupported(players, "intervening-if player filter")
	}
	if err := arity(players, 1); err != nil {
		return nil, err
	}
	if players.Args[0].Kind != Symbol || players.Args[0].Name != "You" {
		return nil, unsupported(players.Args[0], "intervening-if player reference")
	}
	condition := n.Args[1]
	if condition.Name != "PermanentPassesFilter" {
		return nil, unsupported(condition, "intervening-if condition")
	}
	if err := arity(condition, 2); err != nil {
		return nil, err
	}
	if condition.Args[0].Kind != Symbol || condition.Args[0].Name != "ThisPermanent" {
		return nil, unsupported(condition.Args[0], "intervening-if source")
	}
	if condition.Args[1].Kind != Symbol || condition.Args[1].Name != "IsTapped" {
		return nil, unsupported(condition.Args[1], "intervening-if filter")
	}
	plan, err := compileActions(n.Args[2], ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		actions := plan.build()
		trigger := mage.At(mage.YourUpkeep, mage.Sequence(actions.effects...), mage.WithInterveningIf(mage.PermanentIsTapped(mage.This))).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}
