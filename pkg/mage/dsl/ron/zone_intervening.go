package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileZoneIntervening(n Node, ctx scope) (optionFactory, error) {
	event, condition := n.Args[0], n.Args[1]
	if event.Name == "WhenAPermanentIsPutIntoAPlayersGraveyard" {
		if condition.Name != "PermanentPutInGraveyardPassesFilter" {
			return nil, unsupported(condition, "death intervening-if condition")
		}
		if err := arity(condition, 1); err != nil {
			return nil, err
		}
		if condition.Args[0].Kind != Symbol || condition.Args[0].Name != "WasntSacrificed" {
			return nil, unsupported(condition.Args[0], "death cause filter")
		}
		return compileGraveyardTriggerWithOptions(event, n.Args[2], ctx, mage.WithDeathNotSacrificed())
	}
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	subject := event.Args[0]
	if subject.Name != "SinglePermanent" {
		return nil, unsupported(subject, "entry intervening-if subject")
	}
	if err := arity(subject, 1); err != nil {
		return nil, err
	}
	if subject.Args[0].Kind != Symbol || subject.Args[0].Name != "ThisPermanent" {
		return nil, unsupported(subject.Args[0], "entry intervening-if source")
	}
	conditionPlan, err := compileEntryInterveningCondition(condition, ctx)
	if err != nil {
		return nil, err
	}
	ctx.enteringSource = true
	plan, err := compileActions(n.Args[2], ctx)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		bindings := instanceBindings{}
		actions := plan.buildWithBindings(bindings)
		trigger := mage.WhenEvent(mage.Enters(mage.This), mage.Sequence(actions.effects...), mage.WithInterveningIf(conditionPlan(bindings))).(*mage.GenericTriggered)
		actions.effects = nil
		return mage.WithAbility(TriggerA(trigger, actions))
	}, nil
}

func compileEntryInterveningCondition(condition Node, ctx scope) (func(instanceBindings) mage.StateCondition, error) {
	if condition.Name == "EnteringPermanentPassesFilter" {
		if err := arity(condition, 1); err != nil {
			return nil, err
		}
		property := condition.Args[0]
		if property.Kind != Symbol || property.Name != "OnTheBattlefield" {
			return nil, unsupported(property, "entry presence property")
		}
		return func(instanceBindings) mage.StateCondition {
			return mage.PermanentMatchesFilter(mage.This, mage.AnyPermanent)
		}, nil
	}
	if condition.Name != "PermanentPassesFilter" {
		return nil, unsupported(condition, "entry intervening-if condition")
	}
	if err := arity(condition, 2); err != nil {
		return nil, err
	}
	ref, err := compilePermanentRef(condition.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if ref.kind != permanentSource && ref.kind != permanentHost {
		return nil, unsupported(condition.Args[0], "entry intervening-if reference")
	}
	filterNode := condition.Args[1]
	if filterNode.Name != "HasAbility" {
		return nil, unsupported(filterNode, "entry intervening-if property")
	}
	if err := arity(filterNode, 1); err != nil {
		return nil, err
	}
	keyword, err := compileTemporaryKeyword(filterNode.Args[0])
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.StateCondition {
		return mage.PermanentMatchesFilterWithLKI(ref.build(bindings), mage.HasKeywordFilter(keyword))
	}, nil
}
