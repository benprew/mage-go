package mage

// inspectableTargetCondition requires a complete tree before evaluation.
// A known Boolean branch cannot hide an opaque callback in another branch.
func inspectableTargetCondition(condition StateCondition) bool {
	switch c := condition.(type) {
	case yourTurnCondition, sourceAttackedThisTurnCondition, sourceAttackedOrBlockedThisTurnCondition:
		return true
	case *statePredicate:
		if c == nil {
			return false
		}
		switch c.kind {
		case "all", "any":
			if len(c.children) == 0 {
				return false
			}
		case "not":
			if len(c.children) != 1 {
				return false
			}
		default:
			return false
		}
		for _, child := range c.children {
			if !inspectableTargetCondition(child) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
