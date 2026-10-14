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
		if delayed.damageReceived != nil {
			identities = append(identities, delayed.damageReceived.recipient)
		}
		if delayed.sourceCaptured && stateConditionReadsSourcePermanent(delayed.interveningCondition) {
			identities = append(identities, capturedPermanent{delayed.SourceID, delayed.sourceIncarnation})
		}
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

func (e *permanentChoice) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || permanentSetReadsSourcePermanent(e.subjects)
}

func (e *damageFromEffect) refersToSourcePermanent() bool {
	return damageReadsSourcePermanent(e.source) || damageRecipientReadsSourcePermanent(e.recipient) || resolutionValueReadsSourcePermanent(e.amount) || (e.withPrevention && resolutionValueReadsSourcePermanent(e.prevention))
}

func (e *damageEachEffect) refersToSourcePermanent() bool {
	recipients, mixed := e.recipients.(mixedDamageRecipients)
	players, playerOnly := e.recipients.(playerDamageRecipients)
	return damageReadsSourcePermanent(e.source) || resolutionValueReadsSourcePermanent(e.amount) || (mixed && (playerSetReadsSourcePermanent(recipients.players) || permanentSetReadsSourcePermanent(recipients.permanents))) || (playerOnly && playerSetReadsSourcePermanent(players.players))
}

func (e *playerDamageEachEffect) refersToSourcePermanent() bool {
	return damageReadsSourcePermanent(e.source) || playerSetReadsSourcePermanent(e.subjects) || resolutionValueReadsSourcePermanent(e.amount)
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
	return self || (e.controlHistory && playerRefReadsSourcePermanent(e.controlPlayer)) || effectReadsSourcePermanent(e.body) || (e.hasElse && effectReadsSourcePermanent(e.otherwise))
}

func (e *combatStateBranch) refersToSourcePermanent() bool {
	_, self := e.subject.(sourceRef)
	return self || effectReadsSourcePermanent(e.body)
}

func (e *unlessPaidEffect) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || effectReadsSourcePermanent(e.effect) || (e.twoBranches && effectReadsSourcePermanent(e.otherwise))
}

func (e *resolutionManaPayment) refersToSourcePermanent() bool {
	return playerRefReadsSourcePermanent(e.player) || resolutionValueReadsSourcePermanent(e.amount) || effectReadsSourcePermanent(e.effect) || (e.twoBranches && effectReadsSourcePermanent(e.otherwise))
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
	case entryLifePaidValue:
		subject = v.subject
	case entrySacrificeCountValue:
		subject = v.subject
	case counterResolutionValue:
		subject = v.subject
	case permanentCounterInformation:
		subject = v.subject
	case playerNumberValue:
		return playerRefReadsSourcePermanent(v.player)
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
	case graveyardSetCountValue:
		return playerSetReadsSourcePermanent(v.players)
	case graveyardCardCountValue:
		return playerRefReadsSourcePermanent(v.player)
	case exiledCardCountValue:
		if set, ok := v.subjects.(linkedExileSet); ok {
			return permanentRefReadsSourcePermanent(set.source)
		}
		return false
	case permanentCounterTotal:
		return permanentSetReadsSourcePermanent(v.subjects)
	case permanentCountValue:
		return permanentSetReadsSourcePermanent(v.subjects)
	case leastControlledPermanentCount:
		return playerSetReadsSourcePermanent(v.players) || permanentSetReadsSourcePermanent(v.subjects)
	case leastHandSize:
		return playerSetReadsSourcePermanent(v.players)
	case arithmeticValue:
		return resolutionValueReadsSourcePermanent(v.left) || resolutionValueReadsSourcePermanent(v.right)
	default:
		return false
	}
	_, self := subject.(sourceRef)
	return self
}

func permanentSetReadsSourcePermanent(subjects PermanentSet) bool {
	switch s := subjects.(type) {
	case linkedCreatedTokenSet:
		return permanentRefReadsSourcePermanent(s.source)
	case grantRecipientSet:
		return permanentRefReadsSourcePermanent(s.subject)
	case attachedAurasSet:
		return permanentRefReadsSourcePermanent(s.host)
	case attachmentHostSet:
		return permanentSetReadsSourcePermanent(s.hosts)
	case blockersOfSet:
		return permanentRefReadsSourcePermanent(s.attacker)
	case bandedWithSet:
		return permanentRefReadsSourcePermanent(s.subject)
	case blockedAttackersOfSet:
		return permanentRefReadsSourcePermanent(s.blocker)
	case combatOpponentsSet:
		return permanentRefReadsSourcePermanent(s.subject)
	case turnDamageHistoryPermanents:
		return damageReadsSourcePermanent(s.source) || permanentSetReadsSourcePermanent(s.candidates)
	case reverseTurnDamageHistoryPermanents:
		return permanentRefReadsSourcePermanent(s.recipient) || permanentSetReadsSourcePermanent(s.candidates)
	case paidTapSet:
		return permanentRefReadsSourcePermanent(s.source)
	case blockedThisTurnSet:
		return permanentRefReadsSourcePermanent(s.blocker)
	case differencePermanentSet:
		return permanentSetReadsSourcePermanent(s.subjects) || permanentSetReadsSourcePermanent(s.excluded)
	case excludedPermanentSet:
		return permanentSetReadsSourcePermanent(s.subjects) || permanentRefReadsSourcePermanent(s.excluded)
	case leastPowerComparisonSet:
		return permanentSetReadsSourcePermanent(s.subjects) || permanentSetReadsSourcePermanent(s.compared)
	case chosenColorPermanentSet:
		return permanentRefReadsSourcePermanent(s.reference) || permanentSetReadsSourcePermanent(s.subjects)
	case sharingCardTypesSet:
		return permanentRefReadsSourcePermanent(s.reference) || permanentSetReadsSourcePermanent(s.subjects)
	case lowestPowerSet:
		return permanentSetReadsSourcePermanent(s.subjects)
	case highestManaValueSet:
		return permanentSetReadsSourcePermanent(s.subjects)
	case playerSetPermanents:
		return playerSetReadsSourcePermanent(s.players)
	case controlledByPermanentSet:
		return playerRefReadsSourcePermanent(s.controller)
	case ownedPermanentSet:
		return playerRefReadsSourcePermanent(s.owner)
	case filteredPermanentSet:
		return permanentSetReadsSourcePermanent(s.subjects)
	case composedPermanentSet:
		return slices.ContainsFunc(s.members, permanentSetReadsSourcePermanent)
	case manaValueAtMostSet:
		return permanentSetReadsSourcePermanent(s.subjects) || resolutionValueReadsSourcePermanent(s.limit)
	default:
		return false
	}
}

func (e *tapEachEffect) refersToSourcePermanent() bool {
	return permanentSetReadsSourcePermanent(e.subjects)
}

func (e *untapEachEffect) refersToSourcePermanent() bool {
	return permanentSetReadsSourcePermanent(e.subjects)
}

func (e *returnEachToHandEffect) refersToSourcePermanent() bool {
	return permanentSetReadsSourcePermanent(e.subjects)
}

func (e *destroyEachEffect) refersToSourcePermanent() bool {
	return permanentSetReadsSourcePermanent(e.subjects)
}

func (e *exileSubjectEffect) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.subject)
}

func (e *exileEachEffect) refersToSourcePermanent() bool {
	return permanentSetReadsSourcePermanent(e.subjects)
}

func (e *skipNextUntapEffect) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.subject)
}

func (e *skipNextUntapsEffect) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.subject)
}

func (e *skipNextUntapEachEffect) refersToSourcePermanent() bool {
	return permanentSetReadsSourcePermanent(e.subjects)
}

func (e *capturedUntapEffect) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.subject) || playerRefReadsSourcePermanent(e.player)
}

func (e *combatPreventionInstallation) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.subject)
}

func (e *combatSourcePreventionInstallation) refersToSourcePermanent() bool {
	return permanentRefReadsSourcePermanent(e.subject) || permanentSetReadsSourcePermanent(e.subjects)
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

func permanentRefReadsSourcePermanent(ref PermanentRef) bool {
	if host, ok := ref.(permanentHostRef); ok {
		return permanentRefReadsSourcePermanent(host.attachment)
	}
	if choice, ok := ref.(playerChosenPermanentRef); ok {
		return playerRefReadsSourcePermanent(choice.player)
	}
	_, self := ref.(sourceRef)
	return self
}

func (e *counterEachEffect) refersToSourcePermanent() bool {
	return permanentSetReadsSourcePermanent(e.subjects) || resolutionValueReadsSourcePermanent(e.amount)
}

func (e *untilEffect) refersToSourcePermanent() bool {
	if e.sourceBound || playerRefReadsSourcePermanent(e.upkeepPlayer) {
		return true
	}
	return slices.ContainsFunc(e.modifications, func(value ContinuousModification) bool {
		if nilDefinition(value) {
			return false
		}
		m := value.modification()
		return m != nil && (permanentRefReadsSourcePermanent(m.subject) ||
			permanentSetReadsSourcePermanent(m.subjects) ||
			playerRefReadsSourcePermanent(m.controller) ||
			resolutionValueReadsSourcePermanent(m.power) ||
			resolutionValueReadsSourcePermanent(m.toughness))
	})
}

func playerSetReadsSourcePermanent(subjects PlayerSet) bool {
	switch s := subjects.(type) {
	case *opponentsRef:
		return s != nil && playerRefReadsSourcePermanent(s.player)
	case libraryRecyclePlayer:
		return playerRefReadsSourcePermanent(s.player)
	case damageHistoryPlayers:
		return damageReadsSourcePermanent(s.source) || playerSetReadsSourcePermanent(s.candidates)
	default:
		return false
	}
}

func (e *drawEachEffect) refersToSourcePermanent() bool {
	return playerSetReadsSourcePermanent(e.subjects) || resolutionValueReadsSourcePermanent(e.amount)
}

func (e *discardEachEffect) refersToSourcePermanent() bool {
	return playerSetReadsSourcePermanent(e.subjects)
}

func (e *gainLifeEachEffect) refersToSourcePermanent() bool {
	return playerSetReadsSourcePermanent(e.subjects) || resolutionValueReadsSourcePermanent(e.amount)
}

func (e *loseLifeEachEffect) refersToSourcePermanent() bool {
	return playerSetReadsSourcePermanent(e.subjects) || resolutionValueReadsSourcePermanent(e.amount)
}

func (e *libraryRecycleEffect) refersToSourcePermanent() bool {
	return playerSetReadsSourcePermanent(e.subjects)
}

func (e *randomPlayerChoice) refersToSourcePermanent() bool {
	return playerSetReadsSourcePermanent(e.subjects)
}

func (e *permanentLoop) refersToSourcePermanent() bool {
	return permanentSetReadsSourcePermanent(e.subjects) || effectReadsSourcePermanent(e.body)
}
