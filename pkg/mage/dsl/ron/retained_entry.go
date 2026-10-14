package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileRetainedEntry(n Node) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	event := n.Args[0]
	if event.Name != "APlayerWouldPutAPermanentOnTheBattlefield" {
		return nil, unsupported(event, "retained entry event")
	}
	if err := arity(event, 2); err != nil {
		return nil, err
	}
	entrant, err := compileEntryTypeFilter(event.Args[1])
	if err != nil {
		return nil, err
	}
	conjunction := event.Args[0]
	if conjunction.Name != "And" {
		return nil, unsupported(conjunction, "retained entry player conjunction")
	}
	if err := arity(conjunction, 1); err != nil {
		return nil, err
	}
	parts, err := list(conjunction.Args[0])
	if err != nil {
		return nil, err
	}
	if len(parts) != 2 {
		return nil, problem(conjunction, "retained entry requires opponent and count clauses")
	}
	var count *Node
	opponent := false
	for i := range parts {
		part := &parts[i]
		if part.Kind == Symbol && part.Name == "Opponent" && !opponent {
			opponent = true
			continue
		}
		if part.Name == "ControlsNum" && count == nil {
			count = part
			continue
		}
		return nil, unsupported(*part, "retained entry player clause")
	}
	if !opponent || count == nil {
		return nil, problem(conjunction, "retained entry requires opponent and count clauses")
	}
	if err := arity(*count, 2); err != nil {
		return nil, err
	}
	counted, err := compileEntryTypeFilter(count.Args[1])
	if err != nil {
		return nil, err
	}
	comparison := count.Args[0]
	if comparison.Name != "GreaterThanOrEqualTo" {
		return nil, unsupported(comparison, "retained entry comparison")
	}
	if err := arity(comparison, 1); err != nil {
		return nil, err
	}
	amount := comparison.Args[0]
	if amount.Name != "TheNumberOfPermanentsOnTheBattlefield" {
		return nil, unsupported(amount, "retained entry count")
	}
	if err := arity(amount, 1); err != nil {
		return nil, err
	}
	minimumFilter, err := compileRetainedEntryCount(amount.Args[0])
	if err != nil {
		return nil, err
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 2 || operations[0].Kind != Symbol || operations[0].Name != "WouldPutPermanentOnBattlefield_PutPermanentOnBattlefield" {
		return nil, problem(n.Args[1], "retained entry requires put then sacrifice")
	}
	action := operations[1]
	if action.Name != "PlayerAction" {
		return nil, unsupported(action, "retained entry operation")
	}
	if err := arity(action, 2); err != nil {
		return nil, err
	}
	if action.Args[0].Kind != Symbol || action.Args[0].Name != "WouldPutAPermanentOnBattlefield_ThatPlayer" {
		return nil, unsupported(action.Args[0], "retained entry actor")
	}
	sacrifice := action.Args[1]
	if sacrifice.Name != "SacrificeAPermanent" {
		return nil, unsupported(sacrifice, "retained entry sacrifice")
	}
	if err := arity(sacrifice, 1); err != nil {
		return nil, err
	}
	if definitionKey(sacrifice.Args[0]) != definitionKey(count.Args[1]) {
		return nil, problem(sacrifice, "retained entry sacrifice must match the actor count filter")
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.PutThenSacrificeFor(mage.OpponentsOf(mage.You), entrant, counted, mage.CountPermanents(mage.PermanentsControlledBy(mage.You, minimumFilter))))
	}, nil
}

func compileRetainedEntryCount(n Node) (mage.PermanentFilter, error) {
	if n.Name != "And" {
		return mage.PermanentFilter{}, unsupported(n, "retained entry source count conjunction")
	}
	if err := arity(n, 1); err != nil {
		return mage.PermanentFilter{}, err
	}
	parts, err := list(n.Args[0])
	if err != nil {
		return mage.PermanentFilter{}, err
	}
	controller := false
	var filters []mage.PermanentFilter
	for _, part := range parts {
		if part.Name == "ControlledByAPlayer" {
			if controller {
				return mage.PermanentFilter{}, problem(part, "duplicate retained entry count controller")
			}
			if err := arity(part, 1); err != nil {
				return mage.PermanentFilter{}, err
			}
			player := part.Args[0]
			if player.Name != "SinglePlayer" {
				return mage.PermanentFilter{}, unsupported(player, "retained entry count player")
			}
			if err := arity(player, 1); err != nil {
				return mage.PermanentFilter{}, err
			}
			if player.Args[0].Kind != Symbol || player.Args[0].Name != "You" {
				return mage.PermanentFilter{}, unsupported(player.Args[0], "retained entry count controller")
			}
			controller = true
			continue
		}
		filter, err := compileEntryTypeFilter(part)
		if err != nil {
			return mage.PermanentFilter{}, err
		}
		filters = append(filters, filter)
	}
	if !controller || len(filters) == 0 {
		return mage.PermanentFilter{}, problem(n, "retained entry count requires source controller and types")
	}
	return mage.And(filters...), nil
}
