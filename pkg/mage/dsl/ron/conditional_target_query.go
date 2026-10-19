package ron

import "github.com/benprew/mage-go/pkg/mage"

func compileSourceTurnHistoryCondition(n Node) (mage.StateCondition, bool, error) {
	if n.Name != "Or" {
		return nil, false, nil
	}
	if err := arity(n, 1); err != nil {
		return nil, true, err
	}
	parts, err := list(n.Args[0])
	if err != nil {
		return nil, true, err
	}
	history := false
	for _, part := range parts {
		if part.Name == "AttackedThisTurn" || part.Name == "BlockedThisTurn" {
			history = true
		}
	}
	if !history {
		return nil, false, nil
	}
	if len(parts) != 2 {
		return nil, true, problem(n, "source turn history requires attacked and blocked operands")
	}
	for _, part := range parts {
		if _, err := symbol(part); err != nil {
			return nil, true, err
		}
	}
	if (parts[0].Name != "AttackedThisTurn" || parts[1].Name != "BlockedThisTurn") &&
		(parts[1].Name != "AttackedThisTurn" || parts[0].Name != "BlockedThisTurn") {
		return nil, true, unsupported(n, "source turn history")
	}
	return mage.SourceAttackedOrBlockedThisTurnCondition(), true, nil
}
