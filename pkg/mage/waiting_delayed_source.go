package mage

// WaitingDelayedPermanentSources reads saved frames, event subjects and source reads.
// Only abilities that still wait to trigger can supply this source category.
func (g *Game) WaitingDelayedPermanentSources() []DamageSource {
	var sources []DamageSource
	for _, delayed := range g.triggers.Delayed() {
		if delayed == nil {
			continue
		}
		var identities []capturedPermanent
		addSourceReference := func(effect Effect) {
			if reference, ok := effect.(delayedSourcePermanentReference); ok && reference.refersToSourcePermanent() && delayed.sourceCaptured {
				identities = append(identities, capturedPermanent{delayed.SourceID, delayed.sourceIncarnation})
			}
		}
		identities = append(identities, capturedPermanent{delayed.MatchEventID, delayed.matchIncarnation})
		for _, effect := range delayed.Effects {
			addSourceReference(effect)
			if references, ok := effect.(stackPermanentBindingReferences); ok {
				identities = append(identities, references.stackPermanentBindings()...)
			}
		}
		for _, mode := range delayed.modes {
			for _, effect := range mode.Effects {
				addSourceReference(effect)
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

type delayedSourcePermanentReference interface {
	refersToSourcePermanent() bool
}

func (e *destroySubjectEffect) refersToSourcePermanent() bool {
	_, ok := e.subject.(sourceRef)
	return ok
}

func (e *sacrificeEffect) refersToSourcePermanent() bool {
	_, ok := e.subject.(sourceRef)
	return ok
}

func (e *returnToHandSubjectEffect) refersToSourcePermanent() bool {
	_, ok := e.subject.(sourceRef)
	return ok
}

func (e *sequenceEffect) refersToSourcePermanent() bool {
	for _, effect := range e.effects {
		if reference, ok := effect.(delayedSourcePermanentReference); ok && reference.refersToSourcePermanent() {
			return true
		}
	}
	return false
}

func (e *capturedDelayedBody) refersToSourcePermanent() bool {
	reference, ok := e.body.(delayedSourcePermanentReference)
	return ok && reference.refersToSourcePermanent()
}
