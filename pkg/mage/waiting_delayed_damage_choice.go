package mage

import "slices"

func (e *damageSourceChoice) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || slices.ContainsFunc(e.sources, damageReadsSourcePermanent)
}
