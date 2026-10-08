package mage

func (e *createTokenCopiesEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || resolutionValueReadsSourcePermanent(e.count)
}

func (e *createTokenCopiesIntoEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || resolutionValueReadsSourcePermanent(e.count)
}

func (e *createOneTokenEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player)
}
