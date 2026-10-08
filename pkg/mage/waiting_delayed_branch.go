package mage

import "slices"

func (e *cardTypeBranch) refersToSourcePermanent() bool {
	return effectReadsSourcePermanent(e.body)
}

func (e *castBranch) refersToSourcePermanent() bool {
	return effectReadsSourcePermanent(e.effect)
}

func (e *permanentEnchantmentSubtypeBranch) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.subject) || effectReadsSourcePermanent(e.body)
}

func (e *positiveBranch) refersToSourcePermanent() bool {
	return resolutionValueReadsSourcePermanent(e.value) || effectReadsSourcePermanent(e.body)
}

func (e *activationHistoryBranch) refersToSourcePermanent() bool {
	// The history key retains the source incarnation, even with an empty history.
	return true
}

func (e *coinFlipEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || slices.ContainsFunc(e.branches, func(branch CoinBranch) bool {
		b, ok := branch.(coinBranch)
		return ok && effectReadsSourcePermanent(b.effect)
	})
}

func (e *stateBranch) refersToSourcePermanent() bool {
	return stateConditionReadsSourcePermanent(e.condition) || effectReadsSourcePermanent(e.effect)
}

func (e *playerSnapshot) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.subject)
}

func stateConditionReadsSourcePermanent(condition StateCondition) bool {
	switch c := condition.(type) {
	case sourceTappedCondition, sourceTappedLKICondition, sourceBlockingCondition,
		sourceAttackedThisTurnCondition, sourceAttackedOrBlockedThisTurnCondition,
		sourceAttackedOrBlockedThisCombatCondition, sourceAttackHistoryCondition,
		sourceDamagedOpponentThisTurnCondition, sourceCounterCondition:
		return true
	case permanentTappedCondition:
		return permanentRefReadsSourcePermanent(c.subject)
	case permanentAttackingCondition:
		return permanentRefReadsSourcePermanent(c.subject)
	case permanentBlockingCondition:
		return permanentRefReadsSourcePermanent(c.subject)
	case permanentFilterCondition:
		return permanentRefReadsSourcePermanent(c.subject)
	case controlledSinceTurnStartCondition:
		return permanentRefReadsSourcePermanent(c.subject)
	case permanentPlayerCondition:
		return permanentRefReadsSourcePermanent(c.subject)
	case permanentLastTurnAttackCondition:
		return permanentRefReadsSourcePermanent(c.subject) || playerRefReadsSourcePermanent(c.player)
	case *statePredicate:
		if c != nil && (c.kind == "all" || c.kind == "any" || c.kind == "not") {
			return slices.ContainsFunc(c.children, stateConditionReadsSourcePermanent)
		}
	}
	return false
}
