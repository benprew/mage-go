package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileFiniteDamageRedirect(n Node, ctx scope) (effectFactory, error) {
	if !ctx.resolving || ctx.enteringSource || ctx.drawReplacementBody {
		return nil, problem(n, "finite redirection requires an ordinary resolving body")
	}
	event := n.Args[0]
	if err := arity(event, 2); err != nil {
		return nil, err
	}
	amount, err := fixedDamageCapAmount(event.Args[0])
	if err != nil {
		return nil, err
	}
	if amount <= 0 {
		return nil, problem(event.Args[0], "supply a positive fixed damage amount")
	}
	recipient := event.Args[1]
	if recipient.Name != "Permanent" {
		return nil, unsupported(recipient, "finite redirection recipient")
	}
	if err := arity(recipient, 1); err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(recipient.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 || operations[0].Name != "DealToPlayerInstead" {
		return nil, problem(n.Args[1], "supply one DealToPlayerInstead operation")
	}
	if err := arity(operations[0], 1); err != nil {
		return nil, err
	}
	player, err := compilePlayerRef(operations[0].Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.RedirectNextDamageToPlayer(subject.build(bindings), player.build(bindings), mage.Fixed(amount))
	}, nil
}
