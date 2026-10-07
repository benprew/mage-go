package ron

import (
	"slices"

	"github.com/benprew/mage-go/pkg/mage"
)

func compileStaticValueLayer(n Node, parts []Node, ctx scope) (optionFactory, error) {
	if n.Name != "PermanentLayerEffect" || len(parts) != 1 {
		return nil, unsupported(n, "singular static value layer")
	}
	subject, err := compileConditionalSubject(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	part := parts[0]
	var power, toughness valueFactory
	base := false
	switch part.Name {
	case "SetPowerAndToughnessBoth":
		base = true
		if err := arity(part, 1); err != nil {
			return nil, err
		}
		power, err = compileStaticPTValue(part.Args[0], ctx)
		toughness = power
	case "AdjustPTXY":
		if err := arity(part, 4); err != nil {
			return nil, err
		}
		if part.Args[0].Kind != Symbol || part.Args[0].Name != "PlusX" || part.Args[1].Kind != Symbol || part.Args[1].Name != "PlusY" {
			return nil, problem(part, "static XY bonuses require PlusX and PlusY")
		}
		power, err = compileStaticPTValue(part.Args[2], ctx)
		if err != nil {
			return nil, err
		}
		toughness, err = compileStaticPTValue(part.Args[3], ctx)
	case "AdjustPTForEach":
		if err := arity(part, 3); err != nil {
			return nil, err
		}
		if part.Args[0].Kind != IntegerNode || part.Args[1].Kind != IntegerNode {
			return nil, problem(part, "static per-member bonuses require integer multipliers")
		}
		var count valueFactory
		count, err = compileStaticPTValue(part.Args[2], ctx)
		if err != nil {
			return nil, err
		}
		power = func(bindings instanceBindings) mage.ResolutionValue {
			return mage.MultiplyValues(mage.Fixed(part.Args[0].Integer), count(bindings))
		}
		toughness = func(bindings instanceBindings) mage.ResolutionValue {
			return mage.MultiplyValues(mage.Fixed(part.Args[1].Integer), count(bindings))
		}
	default:
		return nil, unsupported(part, "static value modification")
	}
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		condition := mage.StateConditionFunc("live static P/T bonus", []mage.StateDependency{mage.StateBattlefield}, func(mage.StateReader, mage.StateContext) bool { return true })
		modification := mage.ModifyPT(subject, power(instanceBindings{}), toughness(instanceBindings{}))
		if base {
			modification = mage.SetBasePowerToughness(subject, power(instanceBindings{}), toughness(instanceBindings{}))
		}
		return mage.WithAbility(mage.Static(mage.While(condition, modification)))
	}, nil
}

func compileStaticPTValue(n Node, ctx scope) (valueFactory, error) {
	switch n.Name {
	case "Integer":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		if n.Args[0].Kind != IntegerNode {
			return nil, problem(n.Args[0], "expected integer")
		}
		return func(instanceBindings) mage.ResolutionValue { return mage.Fixed(n.Args[0].Integer) }, nil
	case "Plus", "Minus":
		if err := arity(n, 2); err != nil {
			return nil, err
		}
		left, err := compileStaticPTValue(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		right, err := compileStaticPTValue(n.Args[1], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.ResolutionValue {
			if n.Name == "Plus" {
				return mage.AddValues(left(bindings), right(bindings))
			}
			return mage.SubtractValues(left(bindings), right(bindings))
		}, nil
	case "ManaValueOfPermanent":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		subject, err := compileConditionalSubject(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func(instanceBindings) mage.ResolutionValue { return mage.ManaValueOf(subject) }, nil
	case "HalfRoundedDown", "HalfRoundedUp", "Twice":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		value, err := compileStaticPTValue(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.ResolutionValue {
			switch n.Name {
			case "HalfRoundedDown":
				return mage.DivideRoundDown(value(bindings), mage.Fixed(2))
			case "HalfRoundedUp":
				return mage.DivideRoundUp(value(bindings), mage.Fixed(2))
			default:
				return mage.MultiplyValues(mage.Fixed(2), value(bindings))
			}
		}, nil
	case "TheNumberOfPermanentsOnTheBattlefield":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		filter := n.Args[0]
		if containsAttachedCount(filter) {
			return compileStaticAttachedAuraCount(filter, ctx)
		}
		if err := validateStaticCountFilter(filter); err != nil {
			return nil, err
		}
		return compileBattlefieldCount(n, ctx)
	default:
		// P/T reads can depend on this bonus in the same sublayer.
		return nil, unsupported(n, "live static P/T value")
	}
}

func containsAttachedCount(n Node) bool {
	if n.Name == "AttachedToPermanent" {
		return true
	}
	return slices.ContainsFunc(n.Args, containsAttachedCount)
}

func compileStaticAttachedAuraCount(n Node, ctx scope) (valueFactory, error) {
	if n.Name != "And" {
		return nil, unsupported(n, "static attached Aura count")
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	parts, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) != 2 {
		return nil, problem(n, "attached Aura count requires an Aura filter and one host")
	}
	var host mage.PermanentRef
	aura := false
	for _, part := range parts {
		switch part.Name {
		case "IsEnchantmentType":
			if err := arity(part, 1); err != nil {
				return nil, err
			}
			if part.Args[0].Kind != Symbol || part.Args[0].Name != "Aura" {
				return nil, unsupported(part, "static attached Aura type")
			}
			aura = true
		case "AttachedToPermanent":
			if err := arity(part, 1); err != nil {
				return nil, err
			}
			host, err = compileConditionalSubject(part.Args[0], ctx)
			if err != nil {
				return nil, err
			}
		default:
			return nil, unsupported(part, "static attached Aura count filter")
		}
	}
	if !aura || host == nil {
		return nil, problem(n, "attached Aura count requires an Aura filter and one host")
	}
	return func(instanceBindings) mage.ResolutionValue { return mage.CountPermanents(mage.AttachedAuras(host)) }, nil
}

func validateStaticCountFilter(n Node) error {
	switch n.Name {
	case "And":
		if err := arity(n, 1); err != nil {
			return err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return err
		}
		if len(parts) == 0 {
			return problem(n, "static count filter requires operands")
		}
		for _, part := range parts {
			if err := validateStaticCountFilter(part); err != nil {
				return err
			}
		}
		return nil
	case "ControlledByAPlayer":
		if err := arity(n, 1); err != nil {
			return err
		}
		player := n.Args[0]
		if player.Name != "SinglePlayer" {
			return unsupported(player, "static count controller")
		}
		if err := arity(player, 1); err != nil {
			return err
		}
		if player.Args[0].Kind != Symbol || player.Args[0].Name != "You" {
			return unsupported(player.Args[0], "static count controller reference")
		}
		return nil
	default:
		_, err := compileStaticConditionFilter(n)
		return err
	}
}
