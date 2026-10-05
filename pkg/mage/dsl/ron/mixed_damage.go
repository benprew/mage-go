package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileMixedDamageRecipients(n Node) (func() mage.DamageRecipientSet, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	parts, err := list(n.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) != 2 {
		return nil, problem(n, "supply one permanent set and one player set")
	}
	var filter mage.PermanentFilter
	permanent, player := false, false
	for _, part := range parts {
		if err := arity(part, 1); err != nil {
			return nil, err
		}
		switch part.Name {
		case "EachPermanent":
			if permanent {
				return nil, problem(part, "duplicate permanent recipient set")
			}
			filter, err = compileFilter(part.Args[0])
			if err != nil {
				return nil, err
			}
			permanent = true
		case "EachPlayer":
			if player {
				return nil, problem(part, "duplicate player recipient set")
			}
			if part.Args[0].Kind != Symbol || part.Args[0].Name != "AnyPlayer" {
				return nil, unsupported(part.Args[0], "damage player set filter")
			}
			player = true
		default:
			return nil, unsupported(part, "mixed damage recipient set")
		}
	}
	if !permanent || !player {
		return nil, problem(n, "supply one permanent set and one player set")
	}
	return func() mage.DamageRecipientSet {
		return mage.DamageRecipients(mage.EachPlayer, mage.MatchingPermanents(filter))
	}, nil
}
