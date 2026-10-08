package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileDrawStepLifeReplacement(n Node) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	event := n.Args[0]
	if event.Name != "PlayerWouldBeginTheirDrawStep" {
		return nil, unsupported(event, "draw-step replacement event")
	}
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	player := event.Args[0]
	if player.Kind != Symbol || player.Name != "You" {
		return nil, unsupported(player, "draw-step replacement player")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 2 {
		return nil, problem(n.Args[1], "supply a skip payment and a paid life gain")
	}
	payment := operations[0]
	if payment.Name != "MayCost" {
		return nil, unsupported(payment, "draw-step skip payment")
	}
	if err := arity(payment, 1); err != nil {
		return nil, err
	}
	if payment.Args[0].Kind != Symbol || payment.Args[0].Name != "SkipThisDrawStep" {
		return nil, unsupported(payment.Args[0], "draw-step skip cost")
	}
	branch := operations[1]
	if branch.Name != "If" {
		return nil, unsupported(branch, "draw-step paid branch")
	}
	if err := arity(branch, 2); err != nil {
		return nil, err
	}
	if branch.Args[0].Kind != Symbol || branch.Args[0].Name != "CostWasPaid" {
		return nil, unsupported(branch.Args[0], "draw-step payment result")
	}
	body, err := list(branch.Args[1])
	if err != nil {
		return nil, err
	}
	if len(body) != 1 {
		return nil, problem(branch.Args[1], "supply one life gain")
	}
	gain := body[0]
	if gain.Name != "GainLife" {
		return nil, unsupported(gain, "draw-step paid action")
	}
	if err := arity(gain, 1); err != nil {
		return nil, err
	}
	amount := gain.Args[0]
	if amount.Name != "Integer" {
		return nil, unsupported(amount, "draw-step life amount")
	}
	if err := arity(amount, 1); err != nil {
		return nil, err
	}
	value := amount.Args[0]
	if value.Kind != IntegerNode || value.Integer < 0 {
		return nil, problem(value, "supply a nonnegative integer life amount")
	}
	return func() mage.CardOption { return mage.WithAbility(mage.Static(mage.SkipDrawStepForLife(value.Integer))) }, nil
}
