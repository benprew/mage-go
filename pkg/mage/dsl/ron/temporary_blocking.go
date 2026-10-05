package ron

import (
	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func compileFixedPowerFilter(n Node) (mage.PermanentFilter, error) {
	if err := arity(n, 1); err != nil {
		return mage.PermanentFilter{}, err
	}
	comparison := n.Args[0]
	if err := arity(comparison, 1); err != nil {
		return mage.PermanentFilter{}, err
	}
	value := comparison.Args[0]
	if value.Name != "Integer" {
		return mage.PermanentFilter{}, unsupported(value, "fixed power limit")
	}
	if err := arity(value, 1); err != nil {
		return mage.PermanentFilter{}, err
	}
	if value.Args[0].Kind != IntegerNode {
		return mage.PermanentFilter{}, problem(value.Args[0], "power limit must be an integer")
	}
	limit := value.Args[0].Integer
	switch comparison.Name {
	case "LessThanOrEqualTo":
		return mage.HasPowerLTE(limit), nil
	case "GreaterThanOrEqualTo":
		return mage.HasPowerGTE(limit), nil
	case "EqualTo":
		return mage.HasPowerEqual(limit), nil
	default:
		return mage.PermanentFilter{}, unsupported(comparison, "power comparison")
	}
}

func compileTemporaryBlocking(n Node, ctx scope) (effectFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	subject, err := compilePermanentRef(n.Args[0], ctx)
	if err != nil {
		return nil, err
	}
	if subject.kind != permanentTarget {
		return nil, unsupported(n.Args[0], "temporary blocking target")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 {
		return nil, unsupported(n.Args[1], "temporary blocking operations")
	}
	if operations[0].Kind != Symbol || operations[0].Name != "CantBeBlocked" {
		return nil, unsupported(operations[0], "temporary blocking operation")
	}
	if n.Args[2].Kind != Symbol || n.Args[2].Name != "UntilEndOfTurn" {
		return nil, unsupported(n.Args[2], "duration")
	}
	return func(bindings instanceBindings) mage.Effect {
		return mage.Until(core.EndOfTurn, mage.CantBeBlocked(subject.build(bindings)))
	}, nil
}
