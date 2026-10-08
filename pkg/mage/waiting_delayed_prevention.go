package mage

func (e *preventionInstallation) refersToSourcePermanent() bool {
	return damageRecipientReadsSourcePermanent(e.recipient) || resolutionValueReadsSourcePermanent(e.amount)
}

func (e *recipientPreventionInstallation) refersToSourcePermanent() bool {
	return damageRecipientReadsSourcePermanent(e.recipient)
}

func (e *nextPermanentSourcePrevention) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.subject) || playerRefReadsSourcePermanent(e.recipient)
}

func (e *exactSourcePreventionInstallation) refersToSourcePermanent() bool {
	return damageReadsSourcePermanent(e.source) || playerRefReadsSourcePermanent(e.recipient)
}
