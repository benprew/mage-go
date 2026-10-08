package mage

func (e *numberSnapshot) refersToSourcePermanent() bool {
	return resolutionValueReadsSourcePermanent(e.value)
}
