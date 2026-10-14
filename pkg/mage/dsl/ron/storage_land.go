package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileStaticUntapRule(n Node, ctx scope) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) == 1 && operations[0].Name == "DoesntUntapDuringControllersUntap" {
		subject, err := compileUntapSubject(n, ctx)
		if err != nil {
			return nil, err
		}
		return func() mage.CardOption {
			return mage.WithAbility(mage.Static(mage.PreventUntapDuringStep(subject, mage.PermanentMatchesFilter(subject, mage.AnyPermanent))))
		}, nil
	}
	return compileSelfUntapRule(n)
}

func compileUntapSubject(n Node, ctx scope) (mage.PermanentRef, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 || operations[0].Kind != Symbol || operations[0].Name != "DoesntUntapDuringControllersUntap" {
		return nil, unsupported(n.Args[1], "static untap operations")
	}
	return compileConditionalSubject(n.Args[0], ctx)
}

func compileSelfUntapRule(n Node) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	source := n.Args[0]
	if source.Kind != Symbol || source.Name != "ThisPermanent" {
		return nil, unsupported(source, "self untap source")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 || operations[0].Kind != Symbol {
		return nil, unsupported(n.Args[1], "self untap operations")
	}
	var attribute core.Attr
	switch operations[0].Name {
	case "MayChooseNotToUntapDuringUntap":
		attribute = core.AttrMayNotUntap
	case "DoesntUntapDuringControllersUntap":
		attribute = core.AttrDoesNotUntap
	default:
		return nil, unsupported(operations[0], "self untap operation")
	}
	return func() mage.CardOption {
		condition := mage.StateConditionFunc("self untap rule", []mage.StateDependency{mage.StateBattlefield}, func(mage.StateReader, mage.StateContext) bool { return true })
		return mage.WithAbility(mage.Static(mage.While(condition, mage.GrantKeywordModification(mage.This, attribute))))
	}, nil
}

func compileTappedUpkeep(n Node, ctx scope) (optionFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	event := n.Args[0]
	if event.Name == "WhenACreatureBlocksACreature" || event.Name == "WhenACreatureBecomesBlockedByACreature" || event.Name == "Or" {
		return compileCombatPairTriggerWithCondition(event, n.Args[2], ctx, &n.Args[1])
	}
	if event.Name == "WhenAPermanentEntersTheBattlefield" || event.Name == "WhenAPermanentIsPutIntoAPlayersGraveyard" {
		return compileZoneIntervening(n, ctx)
	}
	if event.Name == "AtTheEndOfCombat" {
		return compileEndCombatIntervening(n, ctx)
	}
	if event.Name == "AtTheBeginningOfAPlayersEndStep" {
		return compileEndStepIntervening(n, ctx)
	}
	if event.Name == "AtTheBeginningOfAPlayersDrawStep" {
		return compileDrawStepIntervening(n, ctx)
	}
	if event.Name != "AtTheBeginningOfAPlayersUpkeep" {
		return nil, unsupported(event, "intervening-if event")
	}
	if n.Args[1].Name == "PlayerPassesFilter" || n.Args[1].Name == "APlayerPassesFilter" {
		return compileUpkeepPlayerCondition(n, ctx)
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
	if condition.Args[1].Kind != Symbol || (condition.Args[1].Name != "IsTapped" && condition.Args[1].Name != "StartedThisTurnUntapped") {
		return nil, unsupported(condition.Args[1], "intervening-if filter")
	}
	plan, err := compileActions(n.Args[2], ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		actions := plan.build()
		check := mage.PermanentIsTapped(mage.This)
		if condition.Args[1].Name == "StartedThisTurnUntapped" {
			check = mage.PermanentStartedTurnUntapped(mage.This)
		}
		trigger := mage.At(mage.YourUpkeep, mage.Sequence(actions.effects...), mage.WithInterveningIf(check)).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}
