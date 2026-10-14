package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileCharacteristicCounts(rules []Node, creature bool) (optionFactory, error) {
	var power, toughness *Node
	for i := range rules {
		rule := &rules[i]
		if rule.Name != "CDA_Power" && rule.Name != "CDA_Toughness" {
			continue
		}
		if !creature {
			return nil, problem(*rule, "count CDA requires a creature")
		}
		if err := arity(*rule, 1); err != nil {
			return nil, err
		}
		slot := &toughness
		if rule.Name == "CDA_Power" {
			slot = &power
		}
		if *slot != nil {
			return nil, problem(*rule, "duplicate characteristic count declaration")
		}
		*slot = rule
	}
	if power == nil && toughness == nil {
		return nil, nil
	}
	if toughness == nil {
		return nil, unsupported(*power, "power-only characteristic count")
	}
	if power != nil && definitionKey(power.Args[0]) != definitionKey(toughness.Args[0]) {
		return nil, problem(*power, "power and toughness counts must use the same declaration")
	}
	value := toughness.Args[0]
	if value.Name == "Plus" {
		if power == nil {
			return nil, unsupported(value, "toughness-only additive characteristic count")
		}
		return compileAdditiveCharacteristicCount(value)
	}
	if value.Name != "TheNumberOfPermanentsOnTheBattlefield" {
		return nil, unsupported(value, "characteristic count value")
	}
	if err := arity(value, 1); err != nil {
		return nil, err
	}
	if err := validateCharacteristicCountFilter(value.Args[0]); err != nil {
		return nil, err
	}
	plan, err := compileStaticPermanentFilter(value.Args[0])
	if err != nil {
		return nil, err
	}
	if power == nil && !plan.controlled {
		return nil, unsupported(value, "uncontrolled toughness-only characteristic count")
	}
	return func() mage.CardOption {
		var effect mage.ContinuousEffect
		switch {
		case power == nil:
			effect = mage.ToughnessEqualsControlledCount(plan.filter)
		case plan.controlled:
			effect = mage.PTEqualsControlledCount(plan.filter)
		default:
			effect = mage.PTEqualsCount(plan.filter)
		}
		return mage.WithStaticAbility(effect)
	}, nil
}

func validateCharacteristicCountFilter(n Node) error {
	switch n.Name {
	case "And", "Or":
		if err := arity(n, 1); err != nil {
			return err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return err
		}
		if len(parts) == 0 {
			return problem(n, "characteristic count filter requires operands")
		}
		for _, part := range parts {
			if err := validateCharacteristicCountFilter(part); err != nil {
				return err
			}
		}
	case "ControlledByAPlayer", "IsNamed", "IsCardtype", "IsNonCardtype", "IsCreatureType", "IsNonCreatureType", "IsLandType", "IsNonLandType", "IsEnchantmentType", "IsColor", "IsNonColor", "IsSupertype", "IsNonSupertype":
	default:
		// P/T reads can depend on the count that this declaration changes.
		return unsupported(n, "characteristic count filter")
	}
	return nil
}
