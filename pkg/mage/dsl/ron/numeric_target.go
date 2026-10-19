package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileFixedPTFilter(n Node) (mage.PermanentFilter, error) {
	if err := arity(n, 1); err != nil {
		return mage.PermanentFilter{}, err
	}
	pt := n.Args[0]
	if pt.Name != "PT" {
		return mage.PermanentFilter{}, unsupported(pt, "fixed P/T")
	}
	power, toughness, err := compilePT(pt)
	if err != nil {
		return mage.PermanentFilter{}, err
	}
	return mage.And(mage.HasPowerEqual(power), mage.HasToughnessEqual(toughness)), nil
}

func compileFixedToughnessFilter(n Node) (mage.PermanentFilter, error) {
	if err := arity(n, 1); err != nil {
		return mage.PermanentFilter{}, err
	}
	comparison := n.Args[0]
	if err := arity(comparison, 1); err != nil {
		return mage.PermanentFilter{}, err
	}
	value := comparison.Args[0]
	if value.Name != "Integer" {
		return mage.PermanentFilter{}, unsupported(value, "fixed toughness limit")
	}
	if err := arity(value, 1); err != nil {
		return mage.PermanentFilter{}, err
	}
	if value.Args[0].Kind != IntegerNode {
		return mage.PermanentFilter{}, problem(value.Args[0], "toughness limit must be an integer")
	}
	limit := value.Args[0].Integer
	switch comparison.Name {
	case "EqualTo":
		return mage.HasToughnessEqual(limit), nil
	case "LessThanOrEqualTo":
		return mage.Or(mage.Not(mage.HasToughnessGTE(limit)), mage.HasToughnessEqual(limit)), nil
	case "GreaterThanOrEqualTo":
		return mage.HasToughnessGTE(limit), nil
	default:
		return mage.PermanentFilter{}, unsupported(comparison, "toughness comparison")
	}
}

func compileSourceRelativePowerTarget(n Node, ctx scope) (mage.PermanentFilter, []mage.PermanentTargetOption, error) {
	if err := arity(n, 1); err != nil {
		return mage.PermanentFilter{}, nil, err
	}
	comparison := n.Args[0]
	powerLE := n.Name == "PowerIs"
	if powerLE {
		if err := sourcePowerLEShape(n); err != nil {
			return mage.PermanentFilter{}, nil, err
		}
	} else if n.Name != "ToughnessIs" || comparison.Name != "LessThan" {
		return mage.PermanentFilter{}, nil, unsupported(comparison, "source power comparison")
	}
	if err := arity(comparison, 1); err != nil {
		return mage.PermanentFilter{}, nil, err
	}
	value := comparison.Args[0]
	if value.Name != "PowerOfPermanent" {
		return mage.PermanentFilter{}, nil, unsupported(value, "source power")
	}
	if err := arity(value, 1); err != nil {
		return mage.PermanentFilter{}, nil, err
	}
	ref, err := compilePermanentRef(value.Args[0], ctx)
	if err != nil {
		return mage.PermanentFilter{}, nil, err
	}
	if ref.kind != permanentSource {
		return mage.PermanentFilter{}, nil, unsupported(value.Args[0], "target source power reference")
	}
	option := mage.TargetToughnessLessThanSourcePower()
	if powerLE {
		option = mage.TargetPowerLessThanOrEqualToSourcePower()
	}
	return mage.AnyPermanent, []mage.PermanentTargetOption{option}, nil
}

type manaValueXTargetPlan struct{}

func (manaValueXTargetPlan) permanentOption() mage.PermanentTargetOption {
	return mage.TargetManaValueEqualX()
}

func (manaValueXTargetPlan) spellTarget(name string) *mage.SpellTargetBinding {
	return mage.DeclareSpellTargetManaValueX(name)
}

func compileManaValueXRestriction(n Node, ctx scope) (*manaValueXTargetPlan, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	comparison := n.Args[0]
	if comparison.Name != "EqualTo" {
		return nil, unsupported(comparison, "target mana value comparison")
	}
	if err := arity(comparison, 1); err != nil {
		return nil, err
	}
	value := comparison.Args[0]
	if value.Kind != Symbol || value.Name != "ValueX" || !ctx.x {
		return nil, problem(value, "target mana value requires announced X")
	}
	return &manaValueXTargetPlan{}, nil
}
