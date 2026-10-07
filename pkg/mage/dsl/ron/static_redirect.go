package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileStaticDamageRedirect(n Node, ctx scope, condition mage.StateCondition) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	event := n.Args[0]
	if event.Name != "DamageWouldBeDealtByAPermanentToRecipient" && event.Name != "DamageWouldBeDealtByASourceToRecipient" {
		return nil, unsupported(event, "static redirection event")
	}
	if err := arity(event, 2); err != nil {
		return nil, err
	}
	recipient := event.Args[1]
	if recipient.Name != "Player" {
		return nil, unsupported(recipient, "static redirection recipient")
	}
	if err := arity(recipient, 1); err != nil {
		return nil, err
	}
	if recipient.Args[0].Kind != Symbol || recipient.Args[0].Name != "You" {
		return nil, unsupported(recipient.Args[0], "static redirection player")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 || operations[0].Name != "DealToCreatureOrPlaneswalkerInstead" {
		return nil, problem(n.Args[1], "static redirection requires one permanent destination")
	}
	operation := operations[0]
	if err := arity(operation, 1); err != nil {
		return nil, err
	}
	subject, err := compileConditionalSubject(operation.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if event.Name == "DamageWouldBeDealtByASourceToRecipient" {
		filter := event.Args[0]
		if filter.Name != "IsCardtype" {
			return nil, unsupported(filter, "static redirection source type")
		}
		if err := arity(filter, 1); err != nil {
			return nil, err
		}
		kind, err := cardType(filter.Args[0])
		if err != nil {
			return nil, err
		}
		return func() mage.CardOption {
			return mage.WithAbility(mage.Static(mage.RedirectDamageFromCardTypeToPermanentIf(mage.You, subject, kind, condition)))
		}, nil
	}
	from, err := compileStaticRedirectSourceFilter(event.Args[0])
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.Static(mage.RedirectDamageToPermanentIf(mage.You, subject, from, condition)))
	}, nil
}

func compileStaticRedirectSourceFilter(n Node) (mage.PermanentFilter, error) {
	if n.Name == "IsUnblocked" {
		if _, err := symbol(n); err != nil {
			return mage.PermanentFilter{}, err
		}
		return mage.DamageSourceIsUnblocked(), nil
	}
	if n.Name == "And" || n.Name == "Or" {
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return mage.PermanentFilter{}, err
		}
		if len(parts) == 0 {
			return mage.PermanentFilter{}, problem(n, "source filter must not be empty")
		}
		filters := make([]mage.PermanentFilter, len(parts))
		for i, part := range parts {
			filters[i], err = compileStaticRedirectSourceFilter(part)
			if err != nil {
				return mage.PermanentFilter{}, err
			}
		}
		if n.Name == "And" {
			return mage.And(filters...), nil
		}
		return mage.Or(filters...), nil
	}
	switch n.Name {
	case "IsCardtype", "IsNonCardtype", "IsCreatureType", "IsNonCreatureType", "IsLandType", "IsNonLandType", "IsColor", "IsNonColor", "IsSupertype", "IsNonSupertype":
		return compileFilter(n)
	default:
		return mage.PermanentFilter{}, unsupported(n, "static redirection source filter")
	}
}
