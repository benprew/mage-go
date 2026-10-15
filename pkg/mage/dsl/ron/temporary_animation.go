package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileTemporaryArtifactCreature(n Node, subject permanentReferencePlan) ([]func(instanceBindings) mage.ContinuousModification, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	types, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	if len(types) != 2 {
		return nil, problem(n, "temporary type setting supports only an artifact creature")
	}
	seen := map[core.CardType]bool{}
	var builds []func(instanceBindings) mage.ContinuousModification
	for _, node := range types {
		kind, err := cardType(node)
		if err != nil {
			return nil, err
		}
		if (kind != core.TypeArtifact && kind != core.TypeCreature) || seen[kind] {
			return nil, problem(node, "supply Artifact and Creature once each")
		}
		seen[kind] = true
		// An artifact creature keeps its prior types under CR 205.1b.
		builds = append(builds, func(bindings instanceBindings) mage.ContinuousModification {
			return mage.AddCardType(subject.build(bindings), kind)
		})
	}
	return builds, nil
}

func compileTemporaryEqualPT(n Node, ctx scope, subject permanentReferencePlan) (func(instanceBindings) mage.ContinuousModification, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	value, err := compileTemporaryValue(n.Args[0], ctx, subject)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.ContinuousModification {
		amount := value(bindings)
		return mage.SetBasePowerToughness(subject.build(bindings), amount, amount)
	}, nil
}

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
	domain, err := compileTargetSubtypeDomain(n)
	if err != nil {
		return mage.PermanentFilter{}, nil, err
	}
	filter, options, err := compilePermanentTargetFilterBody(n, ctx)
	if err != nil {
		return mage.PermanentFilter{}, nil, err
	}
	return filter, append(options, mage.TargetWithDomain(domain)), nil
}

func compilePermanentTargetFilterBody(n Node, ctx scope) (mage.PermanentFilter, []mage.PermanentTargetOption, error) {
	if n.Name == "PowerIs" && len(n.Args) == 1 && len(n.Args[0].Args) == 1 && n.Args[0].Args[0].Name == "PowerOfPermanent" {
		return compileSourcePowerLETarget(n, ctx)
	}
	if n.Name == "IsCreatureType" {
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, nil, err
		}
		subtype, err := symbol(n.Args[0])
		if err != nil {
			return mage.PermanentFilter{}, nil, err
		}
		return mage.AnyPermanent, []mage.PermanentTargetOption{mage.TargetHasSubtype(subtypeName(subtype))}, nil
	}
	if n.Name == "IsBlockingAttacker" || n.Name == "IsBlockedByDefender" || n.Name == "Or" && containsCombatTarget(n) {
		return compileCombatTarget(n, ctx)
	}
	if n.Name == "Or" && len(n.Args) == 1 && n.Args[0].Kind == List && len(n.Args[0].Args) == 0 {
		return mage.PermanentFilter{}, nil, problem(n, "target disjunction requires operands")
	}
	if n.Name == "ManaValueIs" {
		if err := compileManaValueXRestriction(n, ctx); err != nil {
			return mage.PermanentFilter{}, nil, err
		}
		return mage.AnyPermanent, []mage.PermanentTargetOption{mage.TargetManaValueEqualX()}, nil
	}
	if n.Name == "ToughnessIs" && len(n.Args) == 1 && n.Args[0].Name == "LessThan" {
		return compileSourcePowerTarget(n, ctx)
	}
	if n.Name == "AttachedToAPermanent" {
		return compileAttachmentTargetFilter(n)
	}
	if n.Name == "OwnedByAPlayer" {
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, nil, err
		}
		player := n.Args[0]
		if player.Kind == Symbol && player.Name == "Opponent" {
			return mage.AnyPermanent, []mage.PermanentTargetOption{mage.TargetOpponentOwns()}, nil
		}
		if player.Name != "SinglePlayer" {
			return mage.PermanentFilter{}, nil, unsupported(player, "permanent target owner")
		}
		if err := arity(player, 1); err != nil {
			return mage.PermanentFilter{}, nil, err
		}
		if player.Args[0].Kind != Symbol || player.Args[0].Name != "You" {
			return mage.PermanentFilter{}, nil, unsupported(player.Args[0], "permanent target owner")
		}
		return mage.AnyPermanent, []mage.PermanentTargetOption{mage.TargetYouOwn()}, nil
	}
	if n.Name == "ControlledByAPlayer" {
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, nil, err
		}
		player := n.Args[0]
		if player.Name == "ControlsA" {
			filter, err := compileControlsMatchingFilter(player)
			if err != nil {
				return mage.PermanentFilter{}, nil, err
			}
			return mage.ControllerControls(filter), nil, nil
		}
		if player.Kind == Symbol && player.Name == "Opponent" {
			return mage.AnyPermanent, []mage.PermanentTargetOption{mage.TargetOpponentControls()}, nil
		}
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

func compileTemporaryManualPT(pt Node, ctx scope, subject permanentReferencePlan) (func(instanceBindings) mage.ContinuousModification, error) {
	if err := arity(pt, 2); err != nil {
		return nil, err
	}
	power, err := compileTemporaryValue(pt.Args[0], ctx, subject)
	if err != nil {
		return nil, err
	}
	toughness, err := compileTemporaryValue(pt.Args[1], ctx, subject)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.ContinuousModification {
		return mage.SetBasePowerToughness(subject.build(bindings), power(bindings), toughness(bindings))
	}, nil
}
