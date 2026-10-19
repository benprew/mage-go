package ron

import "github.com/benprew/mage-go/pkg/mage"

func compilePermanentInformationValue(n Node, ctx scope) (valueFactory, error) {
	if n.Name != "PowerOfPermanent" && n.Name != "ManaValueOfPermanent" {
		return nil, unsupported(n, "permanent information value")
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	permanent, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.ResolutionValue {
		ref := permanent.build(bindings)
		if n.Name == "PowerOfPermanent" {
			return mage.PowerOfWithLKI(ref)
		}
		return mage.ManaValueOfWithLKI(ref)
	}, nil
}
