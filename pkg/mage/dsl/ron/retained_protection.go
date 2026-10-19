package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileRetainedProtection(n Node, condition mage.StateCondition) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	subjects, err := compileStaticPermanentFilter(n.Args[0])
	if err != nil {
		return nil, err
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) == 0 {
		return nil, problem(n.Args[1], "protection requires operations")
	}
	var aura, control bool
	for _, operation := range operations {
		switch operation.Name {
		case "CantBeEnchantedAndDoesntRemove":
			if aura {
				return nil, problem(operation, "duplicate Aura restriction")
			}
			if err := arity(operation, 1); err != nil {
				return nil, err
			}
			filter := operation.Args[0]
			if filter.Name != "IsEnchantmentType" {
				return nil, unsupported(filter, "retained Aura filter")
			}
			if err := arity(filter, 1); err != nil {
				return nil, err
			}
			if filter.Args[0].Kind != Symbol || filter.Args[0].Name != "Aura" {
				return nil, unsupported(filter.Args[0], "retained Aura subtype")
			}
			aura = true
		case "CantBeGainedControlOf":
			if _, err := symbol(operation); err != nil {
				return nil, err
			}
			if control {
				return nil, problem(operation, "duplicate control restriction")
			}
			control = true
		default:
			return nil, unsupported(operation, "retained protection operation")
		}
	}
	return func() mage.CardOption {
		var rules []mage.StaticRule
		if aura {
			rules = append(rules, mage.PreventNewAuraAttachmentsIf(subjects.build(), condition))
		}
		if control {
			rules = append(rules, mage.PreventControlGainsIf(subjects.build(), condition))
		}
		return mage.WithAbility(mage.Static(rules...))
	}, nil
}
