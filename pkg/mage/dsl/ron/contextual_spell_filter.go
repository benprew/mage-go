package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileContextualSpellFilter(n Node) (mage.SpellFilter, error) {
	switch n.Name {
	case "And", "Or":
		if err := arity(n, 1); err != nil {
			return mage.SpellFilter{}, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return mage.SpellFilter{}, err
		}
		if len(parts) == 0 {
			return mage.SpellFilter{}, problem(n, "spell filter requires at least one operand")
		}
		filters := make([]mage.SpellFilter, len(parts))
		for i, part := range parts {
			filters[i], err = compileContextualSpellFilter(part)
			if err != nil {
				return mage.SpellFilter{}, err
			}
		}
		if n.Name == "And" {
			return mage.AllSpellFilters(filters...), nil
		}
		return mage.AnySpellFilter(filters...), nil
	case "ExceptFor":
		if err := arity(n, 1); err != nil {
			return mage.SpellFilter{}, err
		}
		filter, err := compileContextualSpellFilter(n.Args[0])
		if err != nil {
			return mage.SpellFilter{}, err
		}
		return mage.NotSpellFilter(filter), nil
	case "ControlledByAPlayer":
		if err := arity(n, 1); err != nil {
			return mage.SpellFilter{}, err
		}
		player := n.Args[0]
		if player.Name != "SinglePlayer" {
			return mage.SpellFilter{}, unsupported(player, "spell controller")
		}
		if err := arity(player, 1); err != nil {
			return mage.SpellFilter{}, err
		}
		if player.Args[0].Kind != Symbol || player.Args[0].Name != "You" {
			return mage.SpellFilter{}, unsupported(player.Args[0], "spell controller")
		}
		return mage.SpellControlledByYou(), nil
	case "SharesAColorWithAPermanent":
		if err := arity(n, 1); err != nil {
			return mage.SpellFilter{}, err
		}
		filter, controlled, err := compileAttachmentHostFilter(n.Args[0], true)
		if err != nil {
			return mage.SpellFilter{}, err
		}
		return mage.SpellSharesColorWithPermanents(filter, controlled), nil
	default:
		filter, err := compileSpellFilter(n)
		if err != nil {
			return mage.SpellFilter{}, err
		}
		return mage.SpellCardMatches(filter), nil
	}
}
