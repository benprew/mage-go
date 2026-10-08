package mage

// WaitingDelayedPermanentSources reads saved permanent frames and event subjects.
// Only abilities that still wait to trigger can supply this source category.
func (g *Game) WaitingDelayedPermanentSources() []DamageSource {
	var sources []DamageSource
	for _, delayed := range g.triggers.Delayed() {
		if delayed == nil {
			continue
		}
		var identities []capturedPermanent
		identities = append(identities, capturedPermanent{delayed.MatchEventID, delayed.matchIncarnation})
		for _, effect := range delayed.Effects {
			if references, ok := effect.(stackPermanentBindingReferences); ok {
				identities = append(identities, references.stackPermanentBindings()...)
			}
		}
		for _, mode := range delayed.modes {
			for _, effect := range mode.Effects {
				if references, ok := effect.(stackPermanentBindingReferences); ok {
					identities = append(identities, references.stackPermanentBindings()...)
				}
			}
		}
		sources = g.appendPermanentReferenceSources(sources, identities)
	}
	return sources
}

// ChooseDamageSourceFromWaitingDelayedBindings uses only saved waiting references.
// Other source categories need separate contracts for a general source choice.
func ChooseDamageSourceFromWaitingDelayedBindings(player PlayerRef, filter DamageSourceFilter, binding *DamageSourceBinding) ResolvingEffect {
	return &damageSourceChoice{player: player, filter: filter, binding: binding, waitingDelayedBindings: true}
}
