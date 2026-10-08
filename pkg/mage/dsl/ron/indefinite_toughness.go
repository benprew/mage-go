package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileIndefiniteToughness(n Node, ctx scope, subject permanentReferencePlan) (func(instanceBindings) mage.ContinuousModification, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	value, err := compileIndefiniteToughnessValue(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.ContinuousModification {
		return mage.SetBaseToughness(subject.build(bindings), value(bindings))
	}, nil
}

func compileIndefiniteToughnessValue(n Node, ctx scope) (resolutionValueFactory, error) {
	switch n.Name {
	case "Plus", "Minus":
		if err := arity(n, 2); err != nil {
			return nil, err
		}
		left, err := compileIndefiniteToughnessValue(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		right, err := compileIndefiniteToughnessValue(n.Args[1], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.ResolutionValue {
			if n.Name == "Plus" {
				return mage.AddValues(left(bindings), right(bindings))
			}
			return mage.SubtractValues(left(bindings), right(bindings))
		}, nil
	case "PowerOfPermanent", "ToughnessOfPermanent":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		ref, err := compilePermanentRef(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.ResolutionValue {
			if n.Name == "PowerOfPermanent" {
				return mage.PowerOfWithLKI(ref.build(bindings))
			}
			return mage.ToughnessOfWithLKI(ref.build(bindings))
		}, nil
	case "TheNumberOfGraveyardCards":
		value, err := compileGraveyardCount(n, ctx)
		return resolutionValueFactory(value), err
	case "Integer":
		value, err := compileValue(n, ctx)
		if err != nil {
			return nil, err
		}
		return func(instanceBindings) mage.ResolutionValue { return mage.ValueAtResolution(value) }, nil
	default:
		return nil, unsupported(n, "indefinite toughness value")
	}
}

func compileGraveyardCount(n Node, ctx scope) (valueFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	conjunction := n.Args[0]
	if conjunction.Name != "And" {
		return nil, unsupported(conjunction, "graveyard count filter")
	}
	if err := arity(conjunction, 1); err != nil {
		return nil, err
	}
	parts, err := list(conjunction.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) != 2 {
		return nil, problem(conjunction, "supply one card filter and one graveyard player")
	}
	card, zone := parts[0], parts[1]
	if card.Name == "InAPlayersGraveyard" {
		card, zone = zone, card
	}
	if zone.Name != "InAPlayersGraveyard" {
		return nil, unsupported(zone, "graveyard count zone")
	}
	if err := arity(zone, 1); err != nil {
		return nil, err
	}
	player := zone.Args[0]
	if player.Name != "SinglePlayer" {
		return nil, unsupported(player, "graveyard count player")
	}
	if err := arity(player, 1); err != nil {
		return nil, err
	}
	ref, err := compilePlayerRef(player.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	filter, err := compileHandCardFilter(card)
	if err != nil {
		return nil, err
	}
	return func(bindings instanceBindings) mage.ResolutionValue {
		return mage.CountGraveyardCards(ref.build(bindings), filter)
	}, nil
}
