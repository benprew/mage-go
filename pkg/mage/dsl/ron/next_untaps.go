package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileNextUntaps(n Node, ctx scope) (effectFactory, error) {
	if n.Name == "EachPermanentDoesntUntapDuringControllersNextUntap" {
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		subjects, err := compileContextualCollection(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.Effect { return mage.SkipNextUntapEach(subjects(bindings)) }, nil
	}
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	value := n.Args[1]
	if value.Name != "Integer" {
		return nil, unsupported(value, "untap step count")
	}
	if err := arity(value, 1); err != nil {
		return nil, err
	}
	if value.Args[0].Kind != IntegerNode || value.Args[0].Integer < 0 {
		return nil, problem(value, "supply a nonnegative untap step count")
	}
	count := value.Args[0].Integer
	return func(bindings instanceBindings) mage.Effect {
		return mage.SkipNextUntaps(subject.build(bindings), count)
	}, nil
}
