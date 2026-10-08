package mage

import "slices"

func (e *resolutionChoice) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || slices.ContainsFunc(e.options, func(option ResolutionOption) bool {
		return stateConditionReadsSourcePermanent(option.When) || effectReadsSourcePermanent(option.Body)
	})
}
