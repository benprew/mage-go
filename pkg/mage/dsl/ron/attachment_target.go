package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileAttachmentTargetFilter(n Node) (mage.PermanentFilter, []mage.PermanentTargetOption, error) {
	if err := arity(n, 1); err != nil {
		return mage.PermanentFilter{}, nil, err
	}
	filter, controlled, err := compileAttachmentHostFilter(n.Args[0], true)
	if err != nil {
		return mage.PermanentFilter{}, nil, err
	}
	var options []mage.PermanentTargetOption
	if controlled {
		options = append(options, mage.TargetAttachedToYouControl())
	}
	return mage.IsAttachedTo(filter), options, nil
}

func compileAttachmentHostFilter(n Node, allowController bool) (mage.PermanentFilter, bool, error) {
	if n.Name == "ControlledByAPlayer" {
		if !allowController {
			return mage.PermanentFilter{}, false, unsupported(n, "attachment host controller in a disjunction")
		}
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, false, err
		}
		player := n.Args[0]
		if player.Name != "SinglePlayer" {
			return mage.PermanentFilter{}, false, unsupported(player, "attachment host controller")
		}
		if err := arity(player, 1); err != nil {
			return mage.PermanentFilter{}, false, err
		}
		if player.Args[0].Kind != Symbol || player.Args[0].Name != "You" {
			return mage.PermanentFilter{}, false, unsupported(player.Args[0], "attachment host controller")
		}
		return mage.AnyPermanent, true, nil
	}
	if n.Name == "And" || n.Name == "Or" {
		if err := arity(n, 1); err != nil {
			return mage.PermanentFilter{}, false, err
		}
		parts, err := list(n.Args[0])
		if err != nil {
			return mage.PermanentFilter{}, false, err
		}
		if len(parts) == 0 {
			return mage.PermanentFilter{}, false, problem(n, "attachment host filter requires at least one operand")
		}
		filters := make([]mage.PermanentFilter, len(parts))
		controlled := false
		for i, part := range parts {
			filter, next, err := compileAttachmentHostFilter(part, allowController && n.Name == "And")
			if err != nil {
				return mage.PermanentFilter{}, false, err
			}
			if controlled && next {
				return mage.PermanentFilter{}, false, problem(part, "supply one attachment host controller")
			}
			controlled = controlled || next
			filters[i] = filter
		}
		if n.Name == "And" {
			return mage.And(filters...), controlled, nil
		}
		return mage.Or(filters...), false, nil
	}
	filter, err := compileFilter(n)
	return filter, false, err
}
