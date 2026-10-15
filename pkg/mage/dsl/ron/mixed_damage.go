package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileMixedDamageRecipients(n Node, ctx scope) (func(instanceBindings) mage.DamageRecipientSet, error) {
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
	var permanents permanentSetFactory
	var players playerSetFactory
	for _, part := range parts {
		if err := arity(part, 1); err != nil {
			return nil, err
		}
		switch part.Name {
		case "EachPermanent":
			if permanents != nil {
				return nil, problem(part, "duplicate permanent recipient set")
			}
			permanents, err = compileDamageHistoryPermanents(part.Args[0], ctx)
		case "EachPlayer":
			if players != nil {
				return nil, problem(part, "duplicate player recipient set")
			}
			players, err = compileDamageHistoryPlayers(part.Args[0], ctx)
		default:
			return nil, unsupported(part, "mixed damage recipient set")
		}
		if err != nil {
			return nil, err
		}
	}
	if permanents == nil || players == nil {
		return nil, problem(n, "supply one permanent set and one player set")
	}
	return func(bindings instanceBindings) mage.DamageRecipientSet {
		return mage.DamageRecipients(players(bindings), permanents(bindings))
	}, nil
}

func splitDamageHistoryFilter(n Node) (history *Node, filters []Node, err error) {
	if n.Name == "WasDealtDamageByPermanentThisGame" {
		return &n, nil, nil
	}
	if n.Name != "And" {
		return nil, []Node{n}, nil
	}
	if err := arity(n, 1); err != nil {
		return nil, nil, err
	}
	parts, err := list(n.Args[0])
	if err != nil {
		return nil, nil, err
	}
	if len(parts) == 0 {
		return nil, nil, problem(n, "damage recipient filter requires operands")
	}
	filters = make([]Node, 0, len(parts))
	for _, part := range parts {
		if part.Name == "WasDealtDamageByPermanentThisGame" {
			if history != nil {
				return nil, nil, problem(part, "duplicate damage history predicate")
			}
			saved := part
			history = &saved
		} else {
			filters = append(filters, part)
		}
	}
	return history, filters, nil
}

func compileGameDamageHistorySource(n Node, ctx scope) (damageSourceFactory, error) {
	if !ctx.resolving || ctx.drawReplacementBody {
		return nil, problem(n, "damage history requires an ordinary resolving body")
	}
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	ref, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.DamageSourceRef { return mage.DamageSourceOf(ref.build(bindings)) }, nil
}

func compileDamageHistoryPlayers(n Node, ctx scope) (playerSetFactory, error) {
	history, filters, err := splitDamageHistoryFilter(n)
	if err != nil {
		return nil, err
	}
	if history == nil {
		if n.Kind != Symbol || n.Name != "AnyPlayer" {
			return nil, unsupported(n, "damage player set filter")
		}
		return func(instanceBindings) mage.PlayerSet { return mage.EachPlayer }, nil
	}
	source, err := compileGameDamageHistorySource(*history, ctx)
	if err != nil {
		return nil, err
	}
	candidates := playerSetFactory(func(instanceBindings) mage.PlayerSet { return mage.EachPlayer })
	if len(filters) > 1 {
		return nil, problem(n, "damage history requires at most one player filter")
	}
	if len(filters) == 1 {
		filter := filters[0]
		if filter.Kind != Symbol || (filter.Name != "AnyPlayer" && filter.Name != "Opponent") {
			return nil, unsupported(filter, "damage history player filter")
		}
		candidates, err = compileResolutionPlayerSet(filter, ctx)
		if err != nil {
			return nil, err
		}
	}
	return func(bindings instanceBindings) mage.PlayerSet {
		return mage.PlayersDealtDamageBy(source(bindings), candidates(bindings))
	}, nil
}

func compileDamageHistoryPermanents(n Node, ctx scope) (permanentSetFactory, error) {
	history, filters, err := splitDamageHistoryFilter(n)
	if err != nil {
		return nil, err
	}
	if history == nil {
		filter, err := compileFilter(n)
		if err != nil {
			return nil, err
		}
		return func(instanceBindings) mage.PermanentSet { return mage.MatchingPermanents(filter) }, nil
	}
	source, err := compileGameDamageHistorySource(*history, ctx)
	if err != nil {
		return nil, err
	}
	compiled := make([]mage.PermanentFilter, len(filters))
	for i, filter := range filters {
		compiled[i], err = compileFilter(filter)
		if err != nil {
			return nil, err
		}
	}
	filter := mage.AnyPermanent
	if len(compiled) > 0 {
		filter = mage.And(compiled...)
	}
	return func(bindings instanceBindings) mage.PermanentSet {
		return mage.PermanentsDealtDamageBy(source(bindings), mage.MatchingPermanents(filter))
	}, nil
}
