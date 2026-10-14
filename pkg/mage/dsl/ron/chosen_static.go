package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileStaticPlayerControls(n Node, ctx scope) (mage.StateCondition, error) {
	var player mage.PlayerRef
	if n.Args[0].Kind != Symbol {
		return nil, unsupported(n.Args[0], "static condition player")
	}
	switch n.Args[0].Name {
	case "You":
		player = mage.You
	case "TheChosenPlayer":
		if !ctx.permanent || ctx.opponentEntryChoice == nil {
			return nil, problem(n.Args[0], "static chosen player requires a source entry choice")
		}
		player = mage.ChosenPlayerOf(mage.This)
	default:
		return nil, unsupported(n.Args[0], "static condition player")
	}
	selector := n.Args[1]
	if selector.Name != "ControlsA" && selector.Name != "ControlsNo" {
		return nil, unsupported(selector, "static player condition")
	}
	if err := arity(selector, 1); err != nil {
		return nil, err
	}
	filter, chosen, err := compileStaticChosenColorConjunction(selector.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if !chosen && n.Args[0].Name == "You" && selector.Name == "ControlsNo" {
		return mage.ControlsNone(filter), nil
	}
	var condition = mage.PlayerControls(player, filter)
	if chosen {
		condition = mage.PlayerControlsChosenColor(player, filter)
	}
	if selector.Name == "ControlsNo" {
		condition = mage.NotCondition(condition)
	}
	return condition, nil
}

func compileStaticChosenColorConjunction(n Node, ctx scope) (mage.PermanentFilter, bool, error) {
	if n.Name == "IsColor" && n.Kind == Tuple && len(n.Args) == 1 && n.Args[0].Name == "TheChosenColor" {
		if n.Args[0].Kind != Symbol {
			return mage.PermanentFilter{}, false, unsupported(n.Args[0], "static chosen color")
		}
		if !ctx.permanent || !ctx.colorEntryChoice {
			return mage.PermanentFilter{}, false, problem(n.Args[0], "static chosen color requires a source entry choice")
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
			return mage.PermanentFilter{}, false, problem(n, "static condition filter must not be empty")
		}
		filters := make([]mage.PermanentFilter, len(parts))
		chosen := false
		for i, part := range parts {
			var child bool
			filters[i], child, err = compileStaticChosenColorConjunction(part, ctx)
			if err != nil {
				return mage.PermanentFilter{}, false, err
			}
			chosen = chosen || child
		}
		return mage.And(filters...), chosen, nil
	}
	filter, err := compileStaticConditionFilter(n)
	return filter, false, err
}
