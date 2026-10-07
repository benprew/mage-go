package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileBoundedCounterPlacement(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	parts, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) != 1 {
		return nil, problem(n, "bounded counter placement requires one instruction")
	}
	instruction := parts[0]
	if instruction.Name != "UptoNumberCountersOfTypeOnPermanent" {
		return nil, unsupported(instruction, "bounded counter instruction")
	}
	if err := arity(instruction, 3); err != nil {
		return nil, err
	}
	amount, err := compileInstanceCounterAmount(instruction.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	counter, err := compilePermanentCounterType(instruction.Args[1])
	if err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(instruction.Args[2], ctx)
	if err != nil {
		return nil, err
	}
	restriction := n.Args[1]
	if restriction.Name != "CantCauseTheNumberOfCountersOfTypeOnPermanentToBeGreaterThan" {
		return nil, unsupported(restriction, "counter placement restriction")
	}
	if err := arity(restriction, 3); err != nil {
		return nil, err
	}
	restrictedCounter, err := compilePermanentCounterType(restriction.Args[0])
	if err != nil {
		return nil, err
	}
	restrictedSubject, err := compilePermanentRef(restriction.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	if restrictedCounter != counter || restrictedSubject != subject {
		return nil, problem(restriction, "restriction must match the placed counter and subject")
	}
	limit := restriction.Args[2]
	if err := arity(limit, 1); err != nil {
		return nil, err
	}
	if limit.Name != "Integer" || limit.Args[0].Kind != IntegerNode || limit.Args[0].Integer < 0 || limit.Args[0].Integer > 255 {
		return nil, problem(limit, "counter limit requires an integer literal from zero through 255")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.AddCountersUpTo(subject.build(bindings), counter, amount(bindings), limit.Args[0].Integer)
	}, nil
}
