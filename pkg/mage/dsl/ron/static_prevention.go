package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileStaticPrevention(n Node, ctx scope) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	event := n.Args[0]
	if event.Name != "DamageWouldBeDealtByAPermanentToRecipient" {
		return nil, unsupported(event, "static prevention event")
	}
	if err := arity(event, 2); err != nil {
		return nil, err
	}
	from, err := compileFilter(event.Args[0])
	if err != nil {
		return nil, err
	}
	recipient := event.Args[1]
	if recipient.Name != "Permanent" {
		return nil, unsupported(recipient, "static prevention recipient")
	}
	if err := arity(recipient, 1); err != nil {
		return nil, err
	}
	ref, err := compilePermanentRef(recipient.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if ref.kind != permanentSource {
		return nil, unsupported(recipient.Args[0], "static prevention reference")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 {
		return nil, problem(n.Args[1], "static prevention requires exactly one operation")
	}
	operation := operations[0]
	if operation.Kind != Symbol || operation.Name != "PreventThatDamage" {
		return nil, unsupported(operation, "static prevention operation")
	}
	return func() mage.CardOption {
		return mage.WithStaticAbility(mage.PreventDamageFromTo(from, mage.IsID))
	}, nil
}
