package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileEntryLifePayment(rules []Node, ctx scope, creature bool) (optionFactory, error) {
	var declaration *Node
	entries := 0
	for i := range rules {
		rule := &rules[i]
		if rule.Name != "AsPermanentEnters" {
			continue
		}
		entries++
		if rule.Kind != Tuple || len(rule.Args) != 2 || rule.Args[1].Kind != List {
			continue
		}
		for _, operation := range rule.Args[1].Args {
			if operation.Name != "PayAnyAmountOfLifeUpto" {
				continue
			}
			if declaration != nil {
				return nil, problem(*rule, "duplicate entry life payment")
			}
			declaration = rule
		}
	}
	if declaration == nil {
		return nil, nil
	}
	n := *declaration
	if !ctx.permanent || entries != 1 || n.Args[0].Kind != Symbol || n.Args[0].Name != "ThisPermanent" || len(n.Args[1].Args) != 1 {
		return nil, problem(n, "entry life requires one source permanent entry operation")
	}
	operation := n.Args[1].Args[0]
	if err := arity(operation, 1); err != nil {
		return nil, err
	}
	maximum, err := compileEntryLifeMaximum(operation.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	var power, toughness *Node
	for i := range rules {
		rule := &rules[i]
		if rule.Name != "CDA_Power" && rule.Name != "CDA_Toughness" {
			continue
		}
		if !creature {
			return nil, problem(*rule, "entry life CDA requires a creature")
		}
		if err := arity(*rule, 1); err != nil {
			return nil, err
		}
		if rule.Args[0].Kind != Symbol || rule.Args[0].Name != "TheLifePaid" {
			return nil, unsupported(rule.Args[0], "entry life CDA value")
		}
		slot := &toughness
		if rule.Name == "CDA_Power" {
			slot = &power
		}
		if *slot != nil {
			return nil, problem(*rule, "duplicate entry life CDA")
		}
		*slot = rule
	}
	if (power == nil) != (toughness == nil) {
		return nil, problem(n, "entry life requires both CDA declarations")
	}
	return func() mage.CardOption {
		return func(card *mage.BaseCard) {
			mage.WithAbility(mage.PayLifeUpToAsEnters(maximum(instanceBindings{})))(card)
			if power != nil {
				mage.WithStaticAbility(mage.PTEqualsEntryLifePaid())(card)
			}
		}
	}, nil
}

func compileEntryLifeMaximum(n Node, ctx scope) (valueFactory, error) {
	switch n.Name {
	case "Integer":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		if n.Args[0].Kind != IntegerNode || n.Args[0].Integer < 0 {
			return nil, problem(n, "entry life maximum requires a nonnegative integer")
		}
		return func(instanceBindings) mage.ResolutionValue { return mage.Fixed(n.Args[0].Integer) }, nil
	case "Plus":
		if err := arity(n, 2); err != nil {
			return nil, err
		}
		left, err := compileEntryLifeMaximum(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		right, err := compileEntryLifeMaximum(n.Args[1], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.ResolutionValue {
			return mage.AddValues(left(bindings), right(bindings))
		}, nil
	case "TheNumberOfPermanentsOnTheBattlefield":
		if err := arity(n, 1); err != nil {
			return nil, err
		}
		if err := validateStaticCountFilter(n.Args[0]); err != nil {
			return nil, err
		}
		set, err := compileContextualCollection(n.Args[0], ctx)
		if err != nil {
			return nil, err
		}
		return func(bindings instanceBindings) mage.ResolutionValue { return mage.CountPermanents(set(bindings)) }, nil
	case "TheNumberOfGraveyardCards":
		return compileEntryLifeGraveyardCount(n)
	default:
		return nil, unsupported(n, "entry life maximum")
	}
}

func compileEntryLifeGraveyardCount(n Node) (valueFactory, error) {
	if err := arity(n, 1); err != nil {
		return nil, err
	}
	conjunction := n.Args[0]
	if conjunction.Name != "And" {
		return nil, unsupported(conjunction, "entry graveyard count filter")
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
		return nil, unsupported(zone, "entry graveyard count zone")
	}
	if err := arity(zone, 1); err != nil {
		return nil, err
	}
	player := zone.Args[0]
	opponent := player.Kind == Symbol && player.Name == "Opponent"
	if !opponent {
		if player.Name != "SinglePlayer" {
			return nil, unsupported(player, "entry graveyard count player")
		}
		if err := arity(player, 1); err != nil {
			return nil, err
		}
		if player.Args[0].Kind != Symbol || player.Args[0].Name != "You" {
			return nil, unsupported(player.Args[0], "entry graveyard count player reference")
		}
	}
	filter, err := compileHandCardFilter(card)
	if err != nil {
		return nil, err
	}
	return func(instanceBindings) mage.ResolutionValue {
		if opponent {
			return mage.CountGraveyardCardsAmong(mage.OpponentsOf(mage.You), filter)
		}
		return mage.CountGraveyardCards(mage.You, filter)
	}, nil
}
