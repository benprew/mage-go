package ron

import "github.com/benprew/mage-go/pkg/mage"

type resolutionValueFactory func(instanceBindings) mage.ResolutionValue

func compileTemporaryPower(n Node, subject permanentReferencePlan) (func(instanceBindings) mage.ContinuousModification, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	value := n.Args[0]
	if value.Name != "Integer" {
		return nil, unsupported(value, "temporary base power value")
	}
	if err := arity(value, 1); err != nil {
		return nil, err
	}
	if value.Args[0].Kind != IntegerNode {
		return nil, problem(value.Args[0], "expected integer")
	}
	power := value.Args[0].Integer
	return func(bindings instanceBindings) mage.ContinuousModification {
		return mage.SetBasePower(subject.build(bindings), mage.Fixed(power))
	}, nil
}

func compileTemporaryPTX(n Node, ctx scope, subject permanentReferencePlan) (func(instanceBindings) mage.ContinuousModification, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	value, err := compileTemporaryValue(n.Args[2], ctx, subject)
	if err != nil {
		return nil, err
	}
	power, err := compileTemporaryCoordinate(n.Args[0], value)
	if err != nil {
		return nil, err
	}
	toughness, err := compileTemporaryCoordinate(n.Args[1], value)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.ContinuousModification {
		return mage.ModifyPT(subject.build(bindings), power(bindings), toughness(bindings))
	}, nil
}

func compileTemporaryValue(n Node, ctx scope, subject permanentReferencePlan) (resolutionValueFactory, error) {
	if n.Name == "ManaValueOfPermanent" {
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		ref := subject
		if n.Args[0].Kind != Symbol || n.Args[0].Name != "ActionPermanent" {
			var err error
			ref, err = compilePermanentRef(n.Args[0], ctx)
			if err != nil {
				return nil, err
			}
		}
		if ref != subject {
			return nil, problem(n.Args[0], "temporary value must read the modified permanent")
		}
		return func(bindings instanceBindings) mage.ResolutionValue { return mage.ManaValueOf(ref.build(bindings)) }, nil
	}
	value, err := compileValue(n, ctx)
	if err != nil {
		return nil, err
	}
	return func(_ instanceBindings) mage.ResolutionValue { return mage.ValueAtResolution(value) }, nil
}

func compileTemporaryCoordinate(n Node, value resolutionValueFactory) (resolutionValueFactory, error) {
	if n.Kind == Symbol {
		switch n.Name {
		case "PlusX":
			return value, nil
		case "MinusX":
			return func(bindings instanceBindings) mage.ResolutionValue {
				return mage.MultiplyValues(mage.Fixed(-1), value(bindings))
			}, nil
		case "Zero":
			return func(_ instanceBindings) mage.ResolutionValue { return mage.Fixed(0) }, nil
		}
	}
	if n.Name == "Integer" {
		fixed, err := compileValue(n, scope{})
		if err != nil {
			return nil, err
		}
		return func(_ instanceBindings) mage.ResolutionValue { return mage.ValueAtResolution(fixed) }, nil
	}
	return nil, unsupported(n, "temporary P/T coordinate")
}
