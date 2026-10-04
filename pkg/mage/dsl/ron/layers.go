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
	var filter mage.PermanentFilter
	var controlled bool
	if global {
		filter, controlled, err = compileStaticCreatureFilter(n.Args[0])
		if err != nil {
			return nil, err
		}
		if !requiresCreature(n.Args[0]) {
			return nil, unsupported(n.Args[0], "noncreature global layer filter")
		}
	} else if n.Args[0].Kind != Symbol || n.Args[0].Name != "HostPermanent" || !ctx.aura {
		return nil, unsupported(n.Args[0], "static layer reference")
	}
	var effects []func() mage.ContinuousEffect
	for _, part := range parts {
		switch part.Name {
		case "AdjustPT":
			power, toughness, err := compilePT(part)
			if err != nil {
				return nil, err
			}
			if global {
				if controlled {
					effects = append(effects, func() mage.ContinuousEffect { return mage.BoostControlledCreatures(power, toughness, filter) })
				} else {
					effects = append(effects, func() mage.ContinuousEffect { return mage.BoostAllCreaturesIncludingSelf(power, toughness, filter) })
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
				if keyword, ok := keywords[ability.Name]; ok && ability.Kind == Symbol {
					if global {
						if controlled {
							effects = append(effects, func() mage.ContinuousEffect { return mage.GrantKeywordToControlled(keyword, filter) })
						} else {
							effects = append(effects, func() mage.ContinuousEffect { return mage.GrantKeywordToAllIncludingSource(keyword, filter) })
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

func compileStaticCreatureFilter(n Node) (mage.PermanentFilter, bool, error) {
	if n.Name == "ControlledByAPlayer" {
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, false, err
		}
		player := n.Args[0]
		if player.Name != "SinglePlayer" {
			return mage.PermanentFilter{}, false, unsupported(player, "static controller selector")
		}
		if err := arity(player, 1); err != nil {
			return mage.PermanentFilter{}, false, err
		}
		if player.Args[0].Kind != Symbol || player.Args[0].Name != "You" {
			return mage.PermanentFilter{}, false, unsupported(player.Args[0], "static controller reference")
		}
		return mage.AnyPermanent, true, nil
	}
	if n.Name == "And" {
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, false, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return mage.PermanentFilter{}, false, err
		}
		if len(parts) == 0 {
			return mage.PermanentFilter{}, false, problem(n, "empty static conjunction")
		}
		filters := make([]mage.PermanentFilter, 0, len(parts))
		controlled := false
		for _, part := range parts {
			filter, usesController, err := compileStaticCreatureFilter(part)
			if err != nil {
				return mage.PermanentFilter{}, false, err
			}
			filters = append(filters, filter)
			controlled = controlled || usesController
		}
		return mage.And(filters...), controlled, nil
	}
	filter, err := compileFilter(n)
	return filter, false, err
}
