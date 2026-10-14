package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileEntrySacrifice(rules []Node, permanent, creature bool) (optionFactory, error) {
	var declaration *Node
	entries := 0
	for i := range rules {
		rule := &rules[i]
		if rule.Name != "AsPermanentEnters" {
			continue
		}
		entries++
		if rule.Kind != Tuple || len(rule.Args) != 2 || rule.Args[1].Kind != List {
			continue
		}
		for _, operation := range rule.Args[1].Args {
			if operation.Name == "SacrificeAnyNumberOfPermanents" {
				if declaration != nil {
					return nil, problem(*rule, "duplicate entry sacrifice")
				}
				declaration = rule
			}
		}
	}
	if declaration == nil {
		return nil, nil
	}
	n := *declaration
	if !permanent || entries != 1 || n.Args[0].Kind != Symbol || n.Args[0].Name != "ThisPermanent" || len(n.Args[1].Args) != 1 {
		return nil, problem(n, "entry sacrifice requires one source permanent entry operation")
	}
	operation := n.Args[1].Args[0]
	if err := arity(operation, 1); err != nil {
		return nil, err
	}
	if err := validateEntrySacrificeFilter(operation.Args[0]); err != nil {
		return nil, err
	}
	subjects, err := compileFilter(operation.Args[0])
	if err != nil {
		return nil, err
	}
	counted := mage.AnyPermanent
	var power, toughness *Node
	for i := range rules {
		rule := &rules[i]
		if rule.Name != "CDA_Power" && rule.Name != "CDA_Toughness" {
			continue
		}
		if !creature {
			return nil, problem(*rule, "entry sacrifice CDA requires a creature")
		}
		if err := arity(*rule, 1); err != nil {
			return nil, err
		}
		slot := &toughness
		if rule.Name == "CDA_Power" {
			slot = &power
		}
		if *slot != nil {
			return nil, problem(*rule, "duplicate entry sacrifice CDA")
		}
		*slot = rule
	}
	if power != nil || toughness != nil {
		if power == nil || toughness == nil {
			return nil, problem(n, "entry sacrifice requires both CDA declarations")
		}
		if definitionKey(power.Args[0]) != definitionKey(toughness.Args[0]) {
			return nil, problem(*power, "entry sacrifice CDA declarations must match")
		}
		value := power.Args[0]
		if value.Name != "TheNumberOfPermanentsSacrificedAsPermenantEnteredBattlefield" {
			return nil, unsupported(value, "entry sacrifice CDA value")
		}
		if err := arity(value, 2); err != nil {
			return nil, err
		}
		if value.Args[1].Kind != Symbol || value.Args[1].Name != "ThisPermanent" {
			return nil, unsupported(value.Args[1], "entry sacrifice CDA source")
		}
		if err := validateCharacteristicCountFilter(value.Args[0]); err != nil {
			return nil, err
		}
		if err := validateEntrySacrificeFilter(value.Args[0]); err != nil {
			return nil, err
		}
		counted, err = compileFilter(value.Args[0])
		if err != nil {
			return nil, err
		}
	}
	return func() mage.CardOption {
		return func(card *mage.BaseCard) {
			mage.WithAbility(mage.SacrificeAnyNumberAsEnters(subjects, counted))(card)
			if power != nil {
				mage.WithStaticAbility(mage.PTEqualsEntrySacrificeCount())(card)
			}
		}
	}, nil
}

func validateEntrySacrificeFilter(n Node) error {
	if n.Name == "And" || n.Name == "Or" {
		if err := arity(n, 1); err != nil {
			return err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return err
		}
		if len(parts) == 0 {
			return problem(n, "entry sacrifice filter requires operands")
		}
		for _, part := range parts {
			if err := validateEntrySacrificeFilter(part); err != nil {
				return err
			}
		}
		return nil
	}
	if n.Kind == Symbol && (n.Name == "IsUntapped" || n.Name == "IsTapped") {
		return nil
	}
	if err := validateCharacteristicCountFilter(n); err != nil {
		return err
	}
	var family mage.SubtypeFamily
	switch n.Name {
	case "IsLandType", "IsNonLandType":
		land := n
		land.Name = "IsLandType"
		if _, err := compileLandDependenceFilter(land); err != nil {
			return err
		}
		family = mage.SubtypeLand
	case "IsCreatureType", "IsNonCreatureType":
		family = mage.SubtypeCreature
	case "IsEnchantmentType":
		family = mage.SubtypeEnchantment
	default:
		return nil
	}
	if err := arity(n, 1); err != nil {
		return err
	}
	subtype, err := symbol(n.Args[0])
	if err != nil {
		return err
	}
	if err := mage.ValidateSubtypeFamily(family, subtypeName(subtype)); err != nil {
		return problem(n, err.Error())
	}
	return nil
}
