package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileReferencedPermanentDamage(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	source, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if n.Args[2].Name == "Player" {
		return compilePlayerRecipientDamage(n, ctx, func(bindings instanceBindings) mage.DamageSourceRef {
			return mage.DamageSourceOf(source.build(bindings))
		})
	}
	recipient, err := compileDamagePacketRecipient(n.Args[2], ctx)
	if err != nil {
		return nil, err
	}
	amount, err := compileInstanceValue(n.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.DealDamageFrom(mage.DamageSourceOf(source.build(bindings)), recipient(bindings), amount(bindings))
	}, nil
}
