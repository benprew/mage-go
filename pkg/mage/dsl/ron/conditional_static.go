package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileConditionalStatic(n Node, ctx scope) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	condition, err := compileStaticCondition(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	rules, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	var builds []func() mage.ContinuousModification
	var layer core.Layer
	for _, rule := range rules {
		next, nextLayer, err := compileConditionalModifications(rule, ctx)
		if err != nil {
			return nil, err
		}
		// Mixed layers need one membership check for the complete effect.
		if layer != 0 && layer != nextLayer {
			return nil, unsupported(rule, "mixed conditional layers")
		}
		layer = nextLayer
		builds = append(builds, next...)
	}
	if len(builds) == 0 {
		return nil, problem(n.Args[1], "conditional static rule requires modifications")
	}
	return func() mage.CardOption {
		modifications := make([]mage.ContinuousModification, len(builds))
		for i, build := range builds {
			modifications[i] = build()
		}
		return mage.WithAbility(mage.Static(mage.While(condition, modifications...)))
	}, nil
}

func compileStaticCondition(n Node, ctx scope) (mage.StateCondition, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	switch n.Name {
	case "APlayerPassesFilter":
		if n.Args[0].Kind != Symbol || n.Args[0].Name != "Opponent" {
			return nil, unsupported(n.Args[0], "static condition player filter")
		}
		selector := n.Args[1]
		if selector.Name != "ControlsA" {
			return nil, unsupported(selector, "static opponent condition")
		}
		if err := arity(selector, 1); err != nil {
			return nil, err
		}
		filter, err := compileStaticConditionFilter(selector.Args[0])
		if err != nil {
			return nil, err
		}
		return mage.OpponentControls(filter), nil
	case "PlayerPassesFilter":
		if n.Args[0].Kind != Symbol || n.Args[0].Name != "You" {
			return nil, unsupported(n.Args[0], "static condition player")
		}
		selector := n.Args[1]
		if selector.Name != "ControlsA" && selector.Name != "ControlsNo" {
			return nil, unsupported(selector, "static player condition")
		}
		if err := arity(selector, 1); err != nil {
			return nil, err
		}
		filter, err := compileStaticConditionFilter(selector.Args[0])
		if err != nil {
			return nil, err
		}
		if selector.Name == "ControlsNo" {
			return mage.ControlsNone(filter), nil
		}
		return mage.PlayerControls(mage.You, filter), nil
	case "PermanentPassesFilter":
		subject, err := compileConditionalSubject(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		if n.Args[1].Kind == Symbol && n.Args[1].Name == "IsUntapped" {
			return mage.PermanentIsUntapped(subject), nil
		}
		filter, err := compileStaticConditionFilter(n.Args[1])
		if err != nil {
			return nil, err
		}
		return mage.PermanentMatchesFilter(subject, filter), nil
	default:
		return nil, unsupported(n, "static condition")
	}
}

func compileConditionalSubject(n Node, ctx scope) (mage.PermanentRef, error) {
	ref, err := compilePermanentRef(n, ctx)
	if err != nil {
		return nil, err
	}
	if ref.kind != permanentSource && ref.kind != permanentHost {
		return nil, problem(n, "conditional static subject requires a source or Aura host")
	}
	return ref.build(instanceBindings{}), nil
}

func compileConditionalModifications(n Node, ctx scope) ([]func() mage.ContinuousModification, core.Layer, error) {
	if n.Name != "PermanentLayerEffect" && n.Name != "EachPermanentLayerEffect" {
		return nil, 0, unsupported(n, "conditional static body")
	}
	if err := arity(n, 2); err != nil {
		return nil, 0, err
	}
	global := n.Name == "EachPermanentLayerEffect"
	var subject mage.PermanentRef
	var subjects staticPermanentFilterPlan
	var err error
	if global {
		subjects, err = compileStaticPermanentFilter(n.Args[0])
	} else {
		subject, err = compileConditionalSubject(n.Args[0], ctx)
	}
	if err != nil {
		return nil, 0, err
	}
	parts, err := list(n.Args[1])
	if err != nil {
		return nil, 0, err
	}
	var builds []func() mage.ContinuousModification
	var layer core.Layer
	for _, part := range parts {
		switch part.Name {
		case "AdjustPT":
			if layer == core.LayerAbility {
				return nil, 0, unsupported(part, "mixed conditional layers")
			}
			layer = core.LayerPT
			power, toughness, err := compilePT(part)
			if err != nil {
				return nil, 0, err
			}
			builds = append(builds, func() mage.ContinuousModification {
				if global {
					return mage.ModifyEachPT(subjects.build(), mage.Fixed(power), mage.Fixed(toughness))
				}
				return mage.ModifyPT(subject, mage.Fixed(power), mage.Fixed(toughness))
			})
		case "AddAbility":
			if layer == core.LayerPT {
				return nil, 0, unsupported(part, "mixed conditional layers")
			}
			layer = core.LayerAbility
			if err := arity(part, 1); err != nil {
				return nil, 0, err
			}
			abilities, err := list(part.Args[0])
			if err != nil {
				return nil, 0, err
			}
			if len(abilities) == 0 {
				return nil, 0, problem(part, "conditional keyword list must not be empty")
			}
			for _, ability := range abilities {
				keyword, err := compileTemporaryKeyword(ability)
				if err != nil {
					return nil, 0, err
				}
				builds = append(builds, func() mage.ContinuousModification {
					if global {
						return mage.GrantEachKeyword(subjects.build(), keyword)
					}
					return mage.GrantKeywordModification(subject, keyword)
				})
			}
		default:
			return nil, 0, unsupported(part, "conditional static modification")
		}
	}
	if len(builds) == 0 {
		return nil, 0, problem(n.Args[1], "conditional modification list must not be empty")
	}
	return builds, layer, nil
}

func compileStaticConditionFilter(n Node) (mage.PermanentFilter, error) {
	switch n.Name {
	case "IsNonToken":
		if n.Kind != Symbol {
			return mage.PermanentFilter{}, unsupported(n, "static condition filter")
		}
		return mage.Not(mage.IsToken), nil
	case "And", "Or":
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return mage.PermanentFilter{}, err
		}
		if len(parts) == 0 {
			return mage.PermanentFilter{}, problem(n, "static condition filter must not be empty")
		}
		filters := make([]mage.PermanentFilter, len(parts))
		for i, part := range parts {
			filters[i], err = compileStaticConditionFilter(part)
			if err != nil {
				return mage.PermanentFilter{}, err
			}
		}
		if n.Name == "And" {
			return mage.And(filters...), nil
		}
		return mage.Or(filters...), nil
	case "AnyPermanent", "IsTapped", "IsUntapped", "IsAttacking", "IsBlocking",
		"IsCardtype", "IsNonCardtype", "IsCreatureType", "IsNonCreatureType",
		"IsLandType", "IsNonLandType", "IsEnchantmentType", "IsColor", "IsNonColor",
		"IsSupertype", "IsNonSupertype":
	default:
		// A condition must not read the characteristics that its body changes.
		return mage.PermanentFilter{}, unsupported(n, "static condition filter")
	}
	return compileFilter(n)
}
