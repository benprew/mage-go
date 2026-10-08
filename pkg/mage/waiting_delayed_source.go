package mage

import "slices"

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

func (e *tapSubjectEffect) refersToSourcePermanent() bool {
	_, self := e.subject.(sourceRef)
	return self
}

func (e *untapPermanentEffect) refersToSourcePermanent() bool {
	_, self := e.subject.(sourceRef)
	return self
}

func (e *damageFromEffect) refersToSourcePermanent() bool {
	return damageReadsSourcePermanent(e.source) || damageRecipientReadsSourcePermanent(e.recipient) || resolutionValueReadsSourcePermanent(e.amount)
}

func (e *damageEachEffect) refersToSourcePermanent() bool {
	return damageReadsSourcePermanent(e.source) || resolutionValueReadsSourcePermanent(e.amount)
}

func (e *playerDamageEachEffect) refersToSourcePermanent() bool {
	return damageReadsSourcePermanent(e.source) || resolutionValueReadsSourcePermanent(e.amount)
}

func (e *damagePacketsEffect) refersToSourcePermanent() bool {
	return damageReadsSourcePermanent(e.source) || slices.ContainsFunc(e.packets, func(packet DamagePacket) bool {
		return damageRecipientReadsSourcePermanent(packet.recipient) || resolutionValueReadsSourcePermanent(packet.amount)
	})
}

func damageRecipientReadsSourcePermanent(ref DamageRecipientRef) bool {
	if player, ok := ref.(PlayerRef); ok {
		return playerRefReadsSourcePermanent(player)
	}
	_, self := ref.(sourceRef)
	return self
}

func damageReadsSourcePermanent(ref DamageSourceRef) bool {
	switch source := ref.(type) {
	case sourceRef:
		return true
	case permanentDamageSource:
		_, ok := source.subject.(sourceRef)
		return ok
	default:
		return false
	}
}

func (e *sequenceEffect) refersToSourcePermanent() bool {
	return slices.ContainsFunc(e.effects, effectReadsSourcePermanent)
}

func (e *capturedDelayedBody) refersToSourcePermanent() bool {
	return effectReadsSourcePermanent(e.body)
}

func (e *permanentInformationBranch) refersToSourcePermanent() bool {
	_, self := e.subject.(sourceRef)
	return self || effectReadsSourcePermanent(e.body) || (e.hasElse && effectReadsSourcePermanent(e.otherwise))
}

func (e *combatStateBranch) refersToSourcePermanent() bool {
	_, self := e.subject.(sourceRef)
	return self || effectReadsSourcePermanent(e.body)
}

func (e *unlessPaidEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || effectReadsSourcePermanent(e.effect) || (e.twoBranches && effectReadsSourcePermanent(e.otherwise))
}

func (e *resolutionManaPayment) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || resolutionValueReadsSourcePermanent(e.amount) || effectReadsSourcePermanent(e.effect)
}

func resolutionValueReadsSourcePermanent(value ResolutionValue) bool {
	var subject PermanentRef
	switch v := value.(type) {
	case sourceCounterValue:
		return true
	case permanentResolutionValue:
		subject = v.subject
	case permanentManaValue:
		subject = v.subject
	case permanentInformationValue:
		subject = v.subject
	case counterResolutionValue:
		subject = v.subject
	case permanentCounterInformation:
		subject = v.subject
	case playerLifeTotalValue:
		return playerRefReadsSourcePermanent(v.subject)
	case playerHandSizeValue:
		return playerRefReadsSourcePermanent(v.subject)
	case playerLifeGainedThisTurnValue:
		return playerRefReadsSourcePermanent(v.subject)
	case playerDamageHistoryValue:
		return playerRefReadsSourcePermanent(v.subject)
	case matchingHandCount:
		return playerRefReadsSourcePermanent(v.player)
	case graveyardCardCountValue:
		return playerRefReadsSourcePermanent(v.player)
	case permanentCountValue:
		return countSetReadsSourcePermanent(v.subjects)
	case arithmeticValue:
		return resolutionValueReadsSourcePermanent(v.left) || resolutionValueReadsSourcePermanent(v.right)
	default:
		return false
	}
	_, self := subject.(sourceRef)
	return self
}

func countSetReadsSourcePermanent(subjects PermanentSet) bool {
	switch s := subjects.(type) {
	case controlledByPermanentSet:
		return playerRefReadsSourcePermanent(s.controller)
	case ownedPermanentSet:
		return playerRefReadsSourcePermanent(s.owner)
	case filteredPermanentSet:
		return countSetReadsSourcePermanent(s.subjects)
	case composedPermanentSet:
		return slices.ContainsFunc(s.members, countSetReadsSourcePermanent)
	case manaValueAtMostSet:
		return countSetReadsSourcePermanent(s.subjects) || resolutionValueReadsSourcePermanent(s.limit)
	default:
		return false
	}
}

func (e *resolutionCounterPayment) refersToSourcePermanent() bool {
	_, self := e.subject.(sourceRef)
	return self || playerRefReadsSourcePermanent(e.player) || effectReadsSourcePermanent(e.paid) || effectReadsSourcePermanent(e.unpaid)
}

func (e *sacrificePaymentEffect) refersToSourcePermanent() bool {
	_, self := e.subject.(sourceRef)
	return self || playerRefReadsSourcePermanent(e.player) || effectReadsSourcePermanent(e.body)
}

func (e *resolutionActionPaymentEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || actionPaymentReadsSourcePermanent(e.payment) || effectReadsSourcePermanent(e.paid) || effectReadsSourcePermanent(e.unpaid)
}

func actionPaymentReadsSourcePermanent(payment ResolutionActionPayment) bool {
	instruction, ok := payment.(*resolutionActionPayment)
	if !ok {
		return false
	}
	switch instruction.kind {
	case resolutionGainLife:
		return playerRefReadsSourcePermanent(instruction.player) || resolutionValueReadsSourcePermanent(instruction.amount)
	case resolutionDamage:
		player, _ := instruction.recipient.(PlayerRef)
		return damageReadsSourcePermanent(instruction.source) || damageRecipientReadsSourcePermanent(instruction.recipient) || playerRefReadsSourcePermanent(player) || resolutionValueReadsSourcePermanent(instruction.amount)
	case resolutionDestroy:
		_, self := instruction.permanent.(sourceRef)
		return self
	default:
		return false
	}
}

func effectReadsSourcePermanent(effect Effect) bool {
	reference, ok := effect.(delayedSourcePermanentReference)
	return ok && reference.refersToSourcePermanent()
}

func playerRefReadsSourcePermanent(ref PlayerRef) bool {
	var subject PermanentRef
	switch r := ref.(type) {
	case permanentControllerRef:
		subject = r.subject
	case permanentOwnerRef:
		subject = r.subject
	case permanentInformationController:
		subject = r.subject
	default:
		return false
	}
	_, self := subject.(sourceRef)
	return self
}

func (e *lifeAction) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.subject) || resolutionValueReadsSourcePermanent(e.amount)
}

func (e *drawSubjectEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.subject) || resolutionValueReadsSourcePermanent(e.amount)
}

func (e *revealHandForEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player)
}

func (e *discardMatchingHandEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player)
}

func (e *discardSubjectEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.subject) || resolutionValueReadsSourcePermanent(e.amount)
}

func (e *randomDiscardSubjectEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.subject) || resolutionValueReadsSourcePermanent(e.amount)
}

func (e *counterSubjectEffect) refersToSourcePermanent() bool {
	_, self := e.subject.(sourceRef)
	return self || resolutionValueReadsSourcePermanent(e.amount)
}

func (e *boundedCountersEffect) refersToSourcePermanent() bool {
	_, self := e.subject.(sourceRef)
	return self || resolutionValueReadsSourcePermanent(e.amount)
}

func (e *removeCountersSubjectEffect) refersToSourcePermanent() bool {
	_, self := e.subject.(sourceRef)
	return self || resolutionValueReadsSourcePermanent(e.amount)
}

func (e *removeAllCountersEffect) refersToSourcePermanent() bool {
	_, self := e.subject.(sourceRef)
	return self
}
