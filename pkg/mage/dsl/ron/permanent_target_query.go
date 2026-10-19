package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileInspectableTargetFilter(n Node) (mage.PermanentFilter, bool, error) {
	switch n.Name {
	case "PowerAndToughnessIsEqualTo":
		filter, err := compileFixedPTFilter(n)
		return filter, true, err
	case "ToughnessIs":
		if len(n.Args) == 1 && n.Args[0].Name == "LessThan" {
			return mage.PermanentFilter{}, false, nil
		}
		filter, err := compileFixedToughnessFilter(n)
		return filter, true, err
	case "IsTapped", "IsUntapped", "IsAttacking", "IsntAttacking", "IsBlocking":
		if _, err := symbol(n); err != nil {
			return mage.PermanentFilter{}, true, err
		}
		filter, err := compileFilter(n)
		return filter, true, err
	case "AnyPermanent":
		_, err := symbol(n)
		return mage.AnyPermanent, true, err
	case "IsCardtype", "IsNonCardtype":
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, true, err
		}
		kind, err := cardType(n.Args[0])
		if err != nil {
			return mage.PermanentFilter{}, true, err
		}
		filter := mage.HasCardTypeFilter(kind)
		if n.Name == "IsNonCardtype" {
			filter = mage.Not(filter)
		}
		return filter, true, nil
	case "IsCreatureType", "IsNonCreatureType", "IsLandType", "IsNonLandType", "IsEnchantmentType":
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, true, err
		}
		name, err := symbol(n.Args[0])
		if err != nil {
			return mage.PermanentFilter{}, true, err
		}
		name = subtypeName(name)
		family := mage.SubtypeCreature
		switch n.Name {
		case "IsLandType", "IsNonLandType":
			family = mage.SubtypeLand
		case "IsEnchantmentType":
			family = mage.SubtypeEnchantment
		}
		if err := mage.ValidateSubtypePredicate(family, name); err != nil {
			return mage.PermanentFilter{}, true, problem(n.Args[0], err.Error())
		}
		filter := mage.HasSubType(name)
		if n.Name == "IsNonCreatureType" || n.Name == "IsNonLandType" {
			filter = mage.Not(filter)
		}
		return filter, true, nil
	case "IsColor", "IsNonColor":
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, true, err
		}
		value, err := color(n.Args[0])
		if err != nil {
			return mage.PermanentFilter{}, true, err
		}
		filter := mage.HasColorFilter(value)
		if n.Name == "IsNonColor" {
			filter = mage.Not(filter)
		}
		return filter, true, nil
	case "And", "Or":
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, true, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return mage.PermanentFilter{}, true, err
		}
		if len(parts) == 0 {
			return mage.PermanentFilter{}, true, problem(n, "target domain requires operands")
		}
		filters := make([]mage.PermanentFilter, len(parts))
		for i, part := range parts {
			filter, handled, err := compileInspectableTargetFilter(part)
			if err != nil {
				return mage.PermanentFilter{}, true, err
			}
			if !handled {
				return mage.PermanentFilter{}, false, nil
			}
			filters[i] = filter
		}
		if n.Name == "And" {
			return mage.And(filters...), true, nil
		}
		return mage.Or(filters...), true, nil
	default:
		return mage.PermanentFilter{}, false, nil
	}
}

func compileSourceExclusionTarget(n Node, ctx scope) (mage.PermanentTargetOption, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	ref, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if ref.kind != permanentSource {
		return nil, unsupported(n.Args[0], "target exclusion")
	}
	return mage.TargetOtherThanSource(), nil
}
