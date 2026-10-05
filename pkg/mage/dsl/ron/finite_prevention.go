package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileFinitePrevention(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	event := n.Args[0]
	if event.Name != "NextAmountOfDamageThatWouldBeDealtThisTurnToRecipient" {
		return nil, unsupported(event, "finite prevention event")
	}
	if err := arity(event, 2); err != nil {
		return nil, err
	}
	amount, err := compileValue(event.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if event.Args[0].Name == "Integer" && event.Args[0].Args[0].Integer < 0 {
		return nil, problem(event.Args[0], "prevention amount must be nonnegative")
	}
	recipient, err := compileDamagePacketRecipient(event.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 {
		return nil, problem(n.Args[1], "finite prevention requires exactly one operation")
	}
	if operations[0].Kind != Symbol || operations[0].Name != "PreventThatDamage" {
		return nil, unsupported(operations[0], "finite prevention operation")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.PreventNextDamage(recipient(bindings), mage.ValueAtResolution(amount))
	}, nil
}
