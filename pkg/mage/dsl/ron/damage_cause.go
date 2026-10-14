package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileDamageCausePrevention(n Node, ctx scope) (effectFactory, error) {
	event := n.Args[0]
	if err := arity(event, 3); err != nil {
		return nil, err
	}
	filter := event.Args[0]
	if filter.Name != "TargetsPermanent" {
		return nil, unsupported(filter, "damage cause filter")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(filter.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if event.Args[1].Kind != Symbol || event.Args[1].Name != "AnyDamageSource" {
		return nil, unsupported(event.Args[1], "caused damage source filter")
	}
	recipient := event.Args[2]
	if recipient.Name != "Permanent" {
		return nil, unsupported(recipient, "caused damage recipient")
	}
	if err := arity(recipient, 1); err != nil {
		return nil, err
	}
	target, err := compilePermanentRef(recipient.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if target != subject {
		return nil, problem(recipient, "damage cause target and recipient must use the same permanent reference")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 || operations[0].Kind != Symbol || operations[0].Name != "PreventThatDamage" {
		return nil, problem(n.Args[1], "caused damage prevention requires the symbol PreventThatDamage")
	}
	if n.Args[2].Kind != Symbol || n.Args[2].Name != "UntilEndOfTurn" {
		return nil, unsupported(n.Args[2], "caused damage prevention duration")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.PreventDamageCausedByTargetingThisTurn(subject.build(bindings))
	}, nil
}
