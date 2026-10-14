package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileFilteredEntry(n Node) (optionFactory, error) {
	if err := arity(n, 2); err != nil {
		return nil, err
	}
	event := n.Args[0]
	if event.Name != "APermanentWouldEnterTheBattlefield" {
		return nil, unsupported(event, "static entry event")
	}
	if err := arity(event, 1); err != nil {
		return nil, err
	}
	filter := event.Args[0]
	if filter.Name != "And" {
		return nil, unsupported(filter, "static entry conjunction")
	}
	if err := arity(filter, 1); err != nil {
		return nil, err
	}
	parts, err := list(filter.Args[0])
	if err != nil {
		return nil, err
	}
	var characteristics []mage.PermanentFilter
	var players mage.PlayerSet
	for _, part := range parts {
		if part.Name == "ControlledByAPlayer" {
			if players != nil {
				return nil, problem(part, "static entry requires one controller clause")
			}
			if err := arity(part, 1); err != nil {
				return nil, err
			}
			player := part.Args[0]
			if player.Kind != Symbol || player.Name != "Opponent" {
				return nil, unsupported(player, "static entry player filter")
			}
			players = mage.OpponentsOf(mage.You)
			continue
		}
		characteristic, err := compileEntryTypeFilter(part)
		if err != nil {
			return nil, err
		}
		characteristics = append(characteristics, characteristic)
	}
	if players == nil || len(characteristics) == 0 {
		return nil, problem(filter, "static entry requires a controller clause and type filter")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 || operations[0].Kind != Symbol || operations[0].Name != "EntersTapped" {
		return nil, problem(n.Args[1], "static entry requires one EntersTapped symbol")
	}
	return func() mage.CardOption {
		return mage.WithAbility(mage.EntersTappedFor(players, mage.And(characteristics...)))
	}, nil
}

func compileEntryTypeFilter(n Node) (mage.PermanentFilter, error) {
	if err := arity(n, 1); err != nil {
		return mage.PermanentFilter{}, err
	}
	switch n.Name {
	case "IsCardtype", "IsNonCardtype":
		return compileFilter(n)
	case "And", "Or":
		parts, err := list(n.Args[0])
		if err != nil {
			return mage.PermanentFilter{}, err
		}
		if len(parts) == 0 {
			return mage.PermanentFilter{}, problem(n, "entry type filter must not be empty")
		}
		filters := make([]mage.PermanentFilter, len(parts))
		for i, part := range parts {
			filters[i], err = compileEntryTypeFilter(part)
			if err != nil {
				return mage.PermanentFilter{}, err
			}
		}
		if n.Name == "And" {
			return mage.And(filters...), nil
		}
		return mage.Or(filters...), nil
	default:
		return mage.PermanentFilter{}, unsupported(n, "entry type filter")
	}
}
