package mage

func (e *selectedSourceRedirectInstallation) refersToSourcePermanent() bool {
	return damageReadsSourcePermanent(e.source) || permanentRefReadsSourcePermanent(e.subject) || playerRefReadsSourcePermanent(e.player)
}

func (e *permanentDamageRedirectInstallation) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.source) || playerRefReadsSourcePermanent(e.player) || permanentRefReadsSourcePermanent(e.destination)
}
