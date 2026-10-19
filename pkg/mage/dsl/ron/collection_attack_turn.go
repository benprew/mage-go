package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileCollectionAttackTurn(n Node) (effectFactory, error) {
	if err := arity(n, 3); err != nil {
		return nil, err
	}
	if n.Args[2].Kind != Symbol || n.Args[2].Name != "UntilEndOfTurn" {
		return nil, unsupported(n.Args[2], "collection attack duration")
	}
	operations, err := list(n.Args[1])
	if err != nil {
		return nil, err
	}
	if len(operations) != 1 || operations[0].Kind != Symbol {
		return nil, problem(n.Args[1], "supply one attack rule symbol")
	}
	operation := operations[0]
	subject := n.Args[0]
	if operation.Name == "CantAttack" {
		subjects, err := compileStaticPermanentFilter(subject)
		if err != nil {
			return nil, err
		}
		if subjects.controlled || subjects.excludeSource {
			return nil, unsupported(subject, "global resolving attack filter")
		}
		return func(instanceBindings) mage.Effect {
			return (mage.TurnAttackRuleInstruction{Filter: subjects.filter}).Effect()
		}, nil
	}
	if operation.Name != "MustAttack" {
		return nil, unsupported(operation, "collection attack operation")
	}
	if subject.Name != "And" {
		return nil, unsupported(subject, "controlled resolving attack collection")
	}
	if err := arity(subject, 1); err != nil {
		return nil, err
	}
	parts, err := list(subject.Args[0])
	if err != nil {
		return nil, err
	}
	var filters []mage.PermanentFilter
	var player mage.PlayerRef
	for _, part := range parts {
		if part.Name == "ControlledByPlayer" {
			if err := arity(part, 1); err != nil {
				return nil, err
			}
			if player != nil || part.Args[0].Kind != Symbol || part.Args[0].Name != "TheActivePlayer" {
				return nil, unsupported(part, "resolving attack controller")
			}
			player = mage.ActivePlayer
		} else {
			filter, err := compileFilter(part)
			if err != nil {
				return nil, err
			}
			filters = append(filters, filter)
		}
	}
	if player == nil || len(filters) == 0 {
		return nil, problem(subject, "supply an active player controller and a permanent filter")
	}
	return func(instanceBindings) mage.Effect {
		return (mage.TurnAttackRuleInstruction{Filter: mage.And(filters...), Player: player, Require: true}).Effect()
	}, nil
}
