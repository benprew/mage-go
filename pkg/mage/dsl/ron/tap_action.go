package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileTapAction(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.Effect {
		ref := subject.build(bindings)
		if n.Name == "UntapPermanent" {
			return mage.UntapPermanent(ref)
		}
		return mage.TapSubject(ref)
	}, nil
}
