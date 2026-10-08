package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileTemporaryDamageRedirect(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	event := n.Args[0]
	if event.Name != "DamageWouldBeDealtByPermanentToRecipient" {
		return nil, unsupported(event, "temporary redirection event")
	}
	if err := arity(event, 2); err != nil {
		return nil, err
	}
	source, err := compilePermanentRef(event.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	recipient := event.Args[1]
	if recipient.Name != "Player" {
		return nil, unsupported(recipient, "temporary redirection recipient")
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
		return nil, problem(n.Args[1], "temporary redirection requires exactly one operation")
	}
	operation := operations[0]
	if operation.Name != "DealToCreatureOrPlaneswalkerInstead" {
		return nil, unsupported(operation, "temporary redirection operation")
	}
	if err := arity(operation, 1); err != nil {
		return nil, err
	}
	destination, err := compilePermanentRef(operation.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	duration := n.Args[2]
	if duration.Kind != Symbol || duration.Name != "UntilEndOfTurn" {
		return nil, unsupported(duration, "temporary redirection duration")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.RedirectDamageFromPermanentThisTurn(source.build(bindings), player.build(bindings), destination.build(bindings))
	}, nil
}
