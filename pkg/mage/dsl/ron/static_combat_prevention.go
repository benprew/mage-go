package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileStaticCombatPrevention(event Node, ctx scope) (func() []mage.StaticRule, error) {
	if event.Name == "Or" {
		ref, err := compileCombatPreventionUnion(event, ctx)
		if err != nil {
			return nil, err
		}
		if ref.kind != permanentSource && ref.kind != permanentHost {
			return nil, unsupported(event, "static combat prevention reference")
		}
		return func() []mage.StaticRule {
			subject := ref.build(instanceBindings{})
			return []mage.StaticRule{mage.PreventCombatDamageFrom(subject), mage.PreventCombatDamageTo(subject, mage.AnyPermanent)}
		}, nil
	}
	var subject Node
	var filter mage.PermanentFilter
	outgoing := false
	switch event.Name {
	case "CombatDamageWouldBeDealtByCreature":
		if err := arity(event, 1); err != nil {
			return nil, err
		}
		subject = event.Args[0]
		outgoing = true
	case "CombatDamageWouldBeDealtToRecipient":
		if err := arity(event, 1); err != nil {
			return nil, err
		}
		subject = event.Args[0]
		filter = mage.AnyPermanent
	case "CombatDamageWouldBeDealtByACreatureToRecipient":
		if err := arity(event, 2); err != nil {
			return nil, err
		}
		var err error
		filter, err = compileStaticCombatSourceFilter(event.Args[0])
		if err != nil {
			return nil, err
		}
		filter = mage.And(mage.IsCreature, filter)
		subject = event.Args[1]
	default:
		return nil, unsupported(event, "static combat prevention event")
	}
	if !outgoing {
		if subject.Name != "Permanent" {
			return nil, unsupported(subject, "static combat prevention recipient")
		}
		if err := arity(subject, 1); err != nil {
			return nil, err
		}
		subject = subject.Args[0]
	}
	ref, err := compilePermanentRef(subject, ctx)
	if err != nil {
		return nil, err
	}
	if ref.kind != permanentSource && ref.kind != permanentHost {
		return nil, unsupported(subject, "static combat prevention reference")
	}
	return func() []mage.StaticRule {
		if outgoing {
			return []mage.StaticRule{mage.PreventCombatDamageFrom(ref.build(instanceBindings{}))}
		}
		return []mage.StaticRule{mage.PreventCombatDamageTo(ref.build(instanceBindings{}), filter)}
	}, nil
}

func compileStaticCombatSourceFilter(n Node) (mage.PermanentFilter, error) {
	if n.Kind == Symbol {
		switch n.Name {
		case "IsEnchanted":
			return mage.DamageSourceIsEnchanted(), nil
		case "IsntEnchanted":
			return mage.Not(mage.DamageSourceIsEnchanted()), nil
		case "IsAttacking":
			return mage.DamageSourceIsAttacking(), nil
		case "AnyPermanent", "IsTapped", "IsUntapped":
			return compileFilter(n)
		default:
			return mage.PermanentFilter{}, unsupported(n, "static combat damage source filter")
		}
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
			return mage.PermanentFilter{}, problem(n, "supply at least one source filter")
		}
		filters := make([]mage.PermanentFilter, len(parts))
		for i, part := range parts {
			filters[i], err = compileStaticCombatSourceFilter(part)
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
	case "IsCardtype", "IsNonCardtype", "IsLandType", "IsNonLandType", "IsCreatureType", "IsNonCreatureType", "IsEnchantmentType", "IsColor", "IsNonColor", "IsSupertype", "IsNonSupertype":
		return compileFilter(n)
	default:
		return mage.PermanentFilter{}, unsupported(n, "static combat damage source filter")
	}
}
