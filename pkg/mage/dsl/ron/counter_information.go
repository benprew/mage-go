package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileCounterInformation(n Node, ctx scope) (valueFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	counter, err := compilePermanentCounterType(n.Args[0])
	if err != nil {
		return nil, err
	}
	subject := n.Args[1]
	if subject.Name != "SinglePermanent" {
		return nil, unsupported(subject, "counter information subject")
	}
	if err := arity(subject, 1); err != nil {
		return nil, err
	}
	ref, err := compilePermanentRef(subject.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.ResolutionValue {
		return mage.CountersOnWithLKI(ref.build(bindings), counter)
	}, nil
}
