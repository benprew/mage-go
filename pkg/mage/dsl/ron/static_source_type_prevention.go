package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileStaticSourceTypePrevention(n Node, ctx scope) (optionFactory, error) {
	event := n.Args[0]
	if err := arity(event, 2); err != nil {
		return nil, err
	}
	filter := event.Args[0]
	if filter.Name != "IsCardtype" {
		return nil, unsupported(filter, "static damage source type filter")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	kind, err := cardType(filter.Args[0])
	if err != nil {
		return nil, err
	}
	recipient := event.Args[1]
	if recipient.Name != "Permanent" {
		return nil, unsupported(recipient, "static damage recipient")
	}
	if err := arity(recipient, 1); err != nil {
		return nil, err
	}
	ref, err := compilePermanentRef(recipient.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if ref.kind != permanentSource && ref.kind != permanentHost {
		return nil, unsupported(recipient.Args[0], "static damage recipient reference")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 || operations[0].Kind != Symbol || operations[0].Name != "PreventThatDamage" {
		return nil, problem(n.Args[1], "static prevention requires the symbol PreventThatDamage")
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.Static(mage.PreventDamageFromCardType(ref.build(instanceBindings{}), kind)))
	}, nil
}
