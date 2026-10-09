package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileNextCombatPrevention(n Node, ctx scope) (effectFactory, error) {
	event := n.Args[0]
	if err := arity(event, 2); err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(event.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	recipient := event.Args[1]
	if recipient.Name != "Player" {
		return nil, unsupported(recipient, "next combat prevention recipient")
	}
	if err := arity(recipient, 1); err != nil {
		return nil, err
	}
	player, err := compilePlayerRef(recipient.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 {
		return nil, problem(n.Args[1], "next combat prevention requires exactly one operation")
	}
	operation := operations[0]
	if operation.Name != "PreventAllButSomeOfThatDamage" {
		return nil, unsupported(operation, "next combat prevention operation")
	}
	if err := arity(operation, 1); err != nil {
		return nil, err
	}
	amount, err := compileInstanceValue(operation.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if operation.Args[0].Name == "Integer" && operation.Args[0].Args[0].Integer < 0 {
		return nil, problem(operation.Args[0], "damage amount must be nonnegative")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.PreventNextCombatDamageFromPermanentExcept(subject.build(bindings), player.build(bindings), amount(bindings))
	}, nil
}
