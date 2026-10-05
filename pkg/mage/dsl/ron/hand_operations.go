package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileHandCardFilter(n Node) (mage.CardFilter, error) {
	if err := arity(n, 1); err != nil {
		return mage.CardFilter{}, err
	}
	switch n.Name {
	case "IsCardtype", "IsNonCardtype":
		kind, err := cardType(n.Args[0])
		if err != nil {
			return mage.CardFilter{}, err
		}
		return mage.NewCardFilter(n.Name, func(c mage.Card) bool {
			matches := c.HasType(kind)
			if n.Name == "IsNonCardtype" {
				return !matches
			}
			return matches
		}), nil
	case "IsColor", "IsNonColor":
		color, err := color(n.Args[0])
		if err != nil {
			return mage.CardFilter{}, err
		}
		filter := mage.HasColorCardFilter(color)
		if n.Name == "IsNonColor" {
			return mage.NewCardFilter("not "+filter.Text(), func(c mage.Card) bool { return !filter.Match(c) }), nil
		}
		return filter, nil
	case "And", "Or":
		parts, err := list(n.Args[0])
		if err != nil {
			return mage.CardFilter{}, err
		}
		if len(parts) == 0 {
			return mage.CardFilter{}, problem(n, "hand card filter requires at least one operand")
		}
		filters := make([]mage.CardFilter, len(parts))
		for i, part := range parts {
			filters[i], err = compileHandCardFilter(part)
			if err != nil {
				return mage.CardFilter{}, err
			}
		}
		return mage.NewCardFilter(n.Name, func(c mage.Card) bool {
			for _, filter := range filters {
				matches := filter.Match(c)
				if n.Name == "And" && !matches {
					return false
				}
				if n.Name == "Or" && matches {
					return true
				}
			}
			return n.Name == "And"
		}), nil
	default:
		return mage.CardFilter{}, unsupported(n, "hand card filter")
	}
}

func compileHandPlayerAction(n Node, ctx scope) ([]effectFactory, error) {
	if n.Name == "DiscardHand" {
		if n.Kind != Symbol {
			return nil, problem(n, "DiscardHand requires a symbol")
		}
		return []effectFactory{func(bindings instanceBindings) mage.Effect {
			return mage.DiscardMatchingHand(ctx.player.build(bindings), mage.CardFilter{})
		}}, nil
	}
	if n.Name == "RevealHand" {
		if n.Kind != Symbol {
			return nil, problem(n, "RevealHand requires a symbol")
		}
		return []effectFactory{func(bindings instanceBindings) mage.Effect { return mage.RevealHandFor(ctx.player.build(bindings)) }}, nil
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	filter, err := compileHandCardFilter(n.Args[0])
	if err != nil {
		return nil, err
	}
	return []effectFactory{func(bindings instanceBindings) mage.Effect {
		return mage.DiscardMatchingHand(ctx.player.build(bindings), filter)
	}}, nil
}

func compileHandCount(n Node, ctx scope) (valueFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	filter, err := compileHandCardFilter(n.Args[0])
	if err != nil {
		return nil, err
	}
	player, err := compilePlayerRef(n.Args[1], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.ResolutionValue {
		return mage.CountMatchingHand(player.build(bindings), filter)
	}, nil
}
