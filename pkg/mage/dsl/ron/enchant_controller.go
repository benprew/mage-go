package ron

import "github.com/benprew/mage-go/pkg/mage"

type enchantController uint8

const (
	enchantAnyController enchantController = iota
	enchantYou
	enchantOpponent
)

func compileEnchantPermanent(n Node, ctx scope) (optionFactory, error) {
	if !ctx.aura {
		return nil, unsupported(n, "enchant rule without Aura subtype")
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	var controller enchantController
	filter, err := compileEnchantFilter(n.Args[0], &controller)
	if err != nil {
		return nil, err
	}
	return func() mage.CardOption {
		var target mage.Target
		switch controller {
		case enchantYou:
			target = mage.TargetPermanentYouControl(filter)
		case enchantOpponent:
			target = mage.TargetPermanentOpponentControls(filter)
		default:
			target = mage.TargetPermanent(filter)
		}
		if n.Args[0].Name == "IsCardtype" && len(n.Args[0].Args) == 1 && n.Args[0].Args[0].Kind == Symbol && n.Args[0].Args[0].Name == "Creature" {
			target = mage.CreatureEnchantTarget()
		}
		return mage.WithCastTarget(target)
	}, nil
}

func compileEnchantFilter(n Node, controller *enchantController) (mage.PermanentFilter, error) {
	if n.Name == "ControlledByAPlayer" {
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, err
		}
		player := n.Args[0]
		selection := enchantOpponent
		if player.Kind != Symbol || player.Name != "Opponent" {
			if player.Name != "SinglePlayer" {
				return mage.PermanentFilter{}, unsupported(player, "enchant controller")
			}
			if err := arity(player, 1); err != nil {
				return mage.PermanentFilter{}, err
			}
			if player.Args[0].Kind != Symbol || player.Args[0].Name != "You" {
				return mage.PermanentFilter{}, unsupported(player.Args[0], "enchant controller reference")
			}
			selection = enchantYou
		}
		if *controller != enchantAnyController {
			return mage.PermanentFilter{}, problem(n, "supply only one enchant controller restriction")
		}
		*controller = selection
		return mage.AnyPermanent, nil
	}
	if n.Name == "And" {
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return mage.PermanentFilter{}, err
		}
		if len(parts) == 0 {
			return mage.PermanentFilter{}, problem(n, "enchant conjunction requires operands")
		}
		filters := make([]mage.PermanentFilter, len(parts))
		for i, part := range parts {
			filters[i], err = compileEnchantFilter(part, controller)
			if err != nil {
				return mage.PermanentFilter{}, err
			}
		}
		return mage.And(filters...), nil
	}
	return compileFilter(n)
}
