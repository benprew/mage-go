package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
)

func compileTemporaryType(n Node, subject permanentReferencePlan) (func(instanceBindings) mage.ContinuousModification, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	switch n.Name {
	case "SetPT":
		pt := n.Args[0]
		if pt.Name != "PT" {
			return nil, unsupported(pt, "temporary base values")
		}
		power, toughness, err := compilePT(pt)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.ContinuousModification {
			return mage.SetBasePowerToughness(subject.build(bindings), mage.Fixed(power), mage.Fixed(toughness))
		}, nil
	case "AddCardtype":
		kind, err := cardType(n.Args[0])
		if err != nil {
			return nil, err
		}
		if mage.CardTypeAttr(kind) == 0 {
			return nil, unsupported(n.Args[0], "temporary permanent card type")
		}
		return func(bindings instanceBindings) mage.ContinuousModification {
			return mage.AddCardType(subject.build(bindings), kind)
		}, nil
	default:
		name, err := symbol(n.Args[0])
		if err != nil {
			return nil, err
		}
		name = subtypeName(name)
		if err := mage.ValidateSubtypeFamily(mage.SubtypeCreature, name); err != nil {
			return nil, problem(n.Args[0], err.Error())
		}
		return func(bindings instanceBindings) mage.ContinuousModification {
			if n.Name == "SetCreatureType" {
				return mage.SetSubtypeFamily(subject.build(bindings), mage.SubtypeCreature, name)
			}
			return mage.AddSubtypeFamily(subject.build(bindings), mage.SubtypeCreature, name)
		}, nil
	}
}

func compilePermanentTargetFilter(n Node, ctx scope) (mage.PermanentFilter, []mage.PermanentTargetOption, error) {
	if n.Name == "ControlledByAPlayer" {
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, nil, err
		}
		player := n.Args[0]
		if player.Name != "SinglePlayer" {
			return mage.PermanentFilter{}, nil, unsupported(player, "permanent target controller")
		}
		if err := arity(player, 1); err != nil {
			return mage.PermanentFilter{}, nil, err
		}
		if player.Args[0].Kind != Symbol || player.Args[0].Name != "You" {
			return mage.PermanentFilter{}, nil, unsupported(player.Args[0], "permanent target controller")
		}
		return mage.AnyPermanent, []mage.PermanentTargetOption{mage.TargetYouControl()}, nil
	}
	if n.Name == "Other" {
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, nil, err
		}
		ref, err := compilePermanentRef(n.Args[0], ctx)
		if err != nil {
			return mage.PermanentFilter{}, nil, err
		}
		if ref.kind != permanentSource {
			return mage.PermanentFilter{}, nil, unsupported(n.Args[0], "target exclusion")
		}
		return mage.AnyPermanent, []mage.PermanentTargetOption{mage.TargetOtherThanSource()}, nil
	}
	if n.Name == "And" {
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, nil, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return mage.PermanentFilter{}, nil, err
		}
		var filters []mage.PermanentFilter
		var options []mage.PermanentTargetOption
		for _, part := range parts {
			filter, restrictions, err := compilePermanentTargetFilter(part, ctx)
			if err != nil {
				return mage.PermanentFilter{}, nil, err
			}
			filters = append(filters, filter)
			options = append(options, restrictions...)
		}
		return mage.And(filters...), options, nil
	}
	filter, err := compileFilter(n)
	return filter, nil, err
}
