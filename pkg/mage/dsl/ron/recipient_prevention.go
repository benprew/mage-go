package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileRecipientPrevention(n Node, ctx scope) (effectFactory, error) {
	event := n.Args[0]
	count := 1
	if event.Name == "DamageWouldBeDealtByAPermanentToRecipient" {
		count = 2
	}
	if err := arity(event, count); err != nil {
		return nil, err
	}
	var from mage.PermanentFilter
	if count == 2 {
		var err error
		from, err = compilePreventionSourceFilter(event.Args[0])
		if err != nil {
			return nil, err
		}
	}
	recipient, err := compileDamagePacketRecipient(event.Args[count-1], ctx)
	if err != nil {
		return nil, err
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 {
		return nil, problem(n.Args[1], "recipient prevention requires exactly one operation")
	}
	if operations[0].Kind != Symbol || operations[0].Name != "PreventThatDamage" {
		return nil, unsupported(operations[0], "recipient prevention operation")
	}
	if n.Args[2].Kind != Symbol || n.Args[2].Name != "UntilEndOfTurn" {
		return nil, unsupported(n.Args[2], "recipient prevention duration")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.PreventDamageThisTurn(recipient(bindings), from)
	}, nil
}

func compilePreventionSourceFilter(n Node) (mage.PermanentFilter, error) {
	if n.Name == "IsAttacking" && n.Kind == Symbol {
		return mage.DamageSourceIsAttacking(), nil
	}
	if n.Name == "IsBlocking" || n.Name == "PowerIs" {
		return mage.PermanentFilter{}, unsupported(n, "prevention source filter")
	}
	if n.Name != "And" && n.Name != "Or" {
		return compileFilter(n)
	}
	if err := arity(n, 1); err != nil {
		return mage.PermanentFilter{}, err
	}
	parts, err := list(n.Args[0])
	if err != nil {
		return mage.PermanentFilter{}, err
	}
	if len(parts) == 0 {
		return mage.PermanentFilter{}, problem(n, "prevention source filter requires operands")
	}
	filters := make([]mage.PermanentFilter, len(parts))
	for i, part := range parts {
		filters[i], err = compilePreventionSourceFilter(part)
		if err != nil {
			return mage.PermanentFilter{}, err
		}
	}
	if n.Name == "And" {
		return mage.And(filters...), nil
	}
	return mage.Or(filters...), nil
}
