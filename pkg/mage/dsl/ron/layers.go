package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileStaticLayer(n Node, ctx scope) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	parts, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	global := n.Name == "EachPermanentLayerEffect"
	var subjects staticPermanentFilterPlan
	var general bool
	if global {
		subjects, err = compileStaticPermanentFilter(n.Args[0])
		if err != nil {
			return nil, err
		}
		general = !requiresCreature(n.Args[0])
	} else if n.Args[0].Kind != Symbol || n.Args[0].Name != "HostPermanent" || !ctx.aura {
		return nil, unsupported(n.Args[0], "static layer reference")
	}
	var effects []func() mage.ContinuousEffect
	var modifications []func() mage.ContinuousModification
	for _, part := range parts {
		switch part.Name {
		case "AdjustPT":
			power, toughness, err := compilePT(part)
			if err != nil {
				return nil, err
			}
			if general {
				modifications = append(modifications, func() mage.ContinuousModification {
					return mage.ModifyEachPT(subjects.build(), mage.Fixed(power), mage.Fixed(toughness))
				})
			} else if global {
				if subjects.controlled && subjects.excludeSource {
					effects = append(effects, func() mage.ContinuousEffect {
						return mage.BoostOtherControlledCreatures(power, toughness, subjects.filter)
					})
				} else if subjects.controlled {
					effects = append(effects, func() mage.ContinuousEffect { return mage.BoostControlledCreatures(power, toughness, subjects.filter) })
				} else if subjects.excludeSource {
					effects = append(effects, func() mage.ContinuousEffect {
						return mage.BoostAllCreatures(power, toughness, subjects.filter, core.WhileOnBattlefield)
					})
				} else {
					effects = append(effects, func() mage.ContinuousEffect {
						return mage.BoostAllCreaturesIncludingSelf(power, toughness, subjects.filter)
					})
				}
			} else {
				effects = append(effects, func() mage.ContinuousEffect { return mage.BoostAttached(power, toughness, core.AttachAura) })
			}
		case "AddAbility":
			if err := arity(part, 1); err != nil {
				return nil, err
			}
			abilities, err := list(part.Args[0])
			if err != nil {
				return nil, err
			}
			for _, ability := range abilities {
				keyword, ok := keywords[ability.Name]
				ok = ok && ability.Kind == Symbol
				if ability.Name == "Landwalk" {
					keyword, err = compileLandwalkKeyword(ability)
					if err != nil {
						return nil, err
					}
					ok = true
				}
				if ok {
					if general {
						modifications = append(modifications, func() mage.ContinuousModification {
							return mage.GrantEachKeyword(subjects.build(), keyword)
						})
					} else if global {
						if subjects.controlled && subjects.excludeSource {
							effects = append(effects, func() mage.ContinuousEffect { return mage.GrantKeywordToOtherControlled(keyword, subjects.filter) })
						} else if subjects.controlled {
							effects = append(effects, func() mage.ContinuousEffect { return mage.GrantKeywordToControlled(keyword, subjects.filter) })
						} else if subjects.excludeSource {
							effects = append(effects, func() mage.ContinuousEffect { return mage.GrantKeywordToAll(keyword, subjects.filter) })
						} else {
							effects = append(effects, func() mage.ContinuousEffect { return mage.GrantKeywordToAllIncludingSource(keyword, subjects.filter) })
						}
					} else {
						effects = append(effects, func() mage.ContinuousEffect { return mage.GrantAbilityToAttached(keyword, core.AttachAura) })
					}
					continue
				}
				if !global && ability.Name == "ProtectionAndDoesntRemovePermanents" {
					if err := arity(ability, 2); err != nil {
						return nil, err
					}
					exception := ability.Args[1]
					if err := arity(exception, 1); err != nil {
						return nil, err
					}
					if exception.Name != "SinglePermanent" || exception.Args[0].Kind != Symbol || exception.Args[0].Name != "ThisPermanent" {
						return nil, unsupported(exception, "protection exception")
					}
					colors, err := protectionColors(ability.Args[0])
					if err != nil {
						return nil, err
					}
					for _, color := range colors {
						effects = append(effects, func() mage.ContinuousEffect { return mage.GrantProtectionToAttached(color, core.AttachAura) })
					}
					continue
				}
				return nil, unsupported(ability, "static ability grant")
			}
		default:
			return nil, unsupported(part, "static layer action")
		}
	}
	if general {
		if len(modifications) == 0 {
			return nil, problem(n.Args[1], "empty static modification list")
		}
		return func() mage.CardOption {
			values := make([]mage.ContinuousModification, len(modifications))
			for i, build := range modifications {
				values[i] = build()
			}
			condition := mage.StateConditionFunc("unconditional static layer", []mage.StateDependency{mage.StateBattlefield}, func(mage.StateReader, mage.StateContext) bool { return true })
			return mage.WithAbility(mage.Static(mage.While(condition, values...)))
		}, nil
	}
	return func() mage.CardOption {
		values := make([]mage.ContinuousEffect, len(effects))
		for i, build := range effects {
			values[i] = build()
		}
		return mage.WithStaticAbility(values...)
	}, nil
}

func requiresCreature(n Node) bool {
	if n.Name == "IsCardtype" && len(n.Args) == 1 && n.Args[0].Name == "Creature" {
		return true
	}
	if (n.Name == "And" || n.Name == "Or") && len(n.Args) == 1 && n.Args[0].Kind == List {
		parts := n.Args[0].Args
		if len(parts) == 0 {
			return false
		}
		for _, part := range parts {
			if n.Name == "And" && requiresCreature(part) {
				return true
			}
			if n.Name == "Or" && !requiresCreature(part) {
				return false
			}
		}
		return n.Name == "Or"
	}
	return false
}

type staticPermanentFilterPlan struct {
	filter        mage.PermanentFilter
	controlled    bool
	excludeSource bool
}

func (p staticPermanentFilterPlan) build() mage.PermanentSet {
	var subjects = mage.MatchingPermanents(p.filter)
	if p.controlled {
		subjects = mage.ControlledPermanents(p.filter)
	}
	if p.excludeSource {
		subjects = mage.WithoutPermanent(subjects, mage.This)
	}
	return subjects
}

func compileStaticPermanentFilter(n Node) (staticPermanentFilterPlan, error) {
	if n.Name == "Other" {
		if err := arity(n, 1); err != nil {
			return staticPermanentFilterPlan{}, err
		}
		if n.Args[0].Kind != Symbol || n.Args[0].Name != "ThisPermanent" {
			return staticPermanentFilterPlan{}, unsupported(n.Args[0], "static exclusion reference")
		}
		return staticPermanentFilterPlan{filter: mage.AnyPermanent, excludeSource: true}, nil
	}
	if n.Name == "ControlledByAPlayer" {
		if err := arity(n, 1); err != nil {
			return staticPermanentFilterPlan{}, err
		}
		player := n.Args[0]
		if player.Name != "SinglePlayer" {
			return staticPermanentFilterPlan{}, unsupported(player, "static controller selector")
		}
		if err := arity(player, 1); err != nil {
			return staticPermanentFilterPlan{}, err
		}
		if player.Args[0].Kind != Symbol || player.Args[0].Name != "You" {
			return staticPermanentFilterPlan{}, unsupported(player.Args[0], "static controller reference")
		}
		return staticPermanentFilterPlan{filter: mage.AnyPermanent, controlled: true}, nil
	}
	if n.Name == "And" || n.Name == "Or" {
		if err := arity(n, 1); err != nil {
			return staticPermanentFilterPlan{}, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return staticPermanentFilterPlan{}, err
		}
		if len(parts) == 0 {
			return staticPermanentFilterPlan{}, problem(n, "empty static filter")
		}
		filters := make([]mage.PermanentFilter, 0, len(parts))
		plan := staticPermanentFilterPlan{}
		for _, part := range parts {
			child, err := compileStaticPermanentFilter(part)
			if err != nil {
				return staticPermanentFilterPlan{}, err
			}
			if n.Name == "Or" && (child.controlled || child.excludeSource) {
				return staticPermanentFilterPlan{}, unsupported(part, "source-dependent static disjunction")
			}
			filters = append(filters, child.filter)
			plan.controlled = plan.controlled || child.controlled
			plan.excludeSource = plan.excludeSource || child.excludeSource
		}
		plan.filter = mage.And(filters...)
		if n.Name == "Or" {
			plan.filter = mage.Or(filters...)
		}
		return plan, nil
	}
	filter, err := compileFilter(n)
	return staticPermanentFilterPlan{filter: filter}, err
}
