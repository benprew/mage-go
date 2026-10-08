package mage

func (e *millSubjectEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.subject) || resolutionValueReadsSourcePermanent(e.amount)
}
