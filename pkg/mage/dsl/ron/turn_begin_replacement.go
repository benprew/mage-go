package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileTurnBeginReplacement(n Node, condition mage.StateCondition) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	event := n.Args[0]
	if event.Name != "PlayerWouldBeginTheirTurn" {
		return nil, unsupported(event, "turn replacement event")
	}
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	if event.Args[0].Kind != Symbol || event.Args[0].Name != "You" {
		return nil, unsupported(event.Args[0], "turn replacement player")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 2 {
		return nil, problem(n.Args[1], "turn replacement requires an optional skip and its paid untap")
	}
	payment, branch := operations[0], operations[1]
	if payment.Name != "MayCost" {
		return nil, unsupported(payment, "turn replacement payment")
	}
	if err := arity(payment, 1); err != nil {
		return nil, err
	}
	if payment.Args[0].Kind != Symbol || payment.Args[0].Name != "SkipThisTurn" {
		return nil, unsupported(payment.Args[0], "turn replacement cost")
	}
	if branch.Name != "If" {
		return nil, unsupported(branch, "turn replacement paid branch")
	}
	if err := arity(branch, 2); err != nil {
		return nil, err
	}
	if branch.Args[0].Kind != Symbol || branch.Args[0].Name != "CostWasPaid" {
		return nil, unsupported(branch.Args[0], "turn replacement payment result")
	}
	actions, err := list(branch.Args[1])
	if err != nil {
		return nil, err
	}
	if len(actions) != 1 || actions[0].Name != "UntapPermanent" {
		return nil, problem(branch.Args[1], "paid turn replacement requires one source untap")
	}
	if err := arity(actions[0], 1); err != nil {
		return nil, err
	}
	if actions[0].Args[0].Kind != Symbol || actions[0].Args[0].Name != "ThisPermanent" {
		return nil, unsupported(actions[0].Args[0], "turn replacement untap subject")
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.Static(mage.MaySkipYourTurnToUntap(mage.This, condition)))
	}, nil
}
