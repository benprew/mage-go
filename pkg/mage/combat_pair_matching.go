package mage

import (
	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type filteredCombatPairPattern struct {
	subject          PermanentRef
	filter           PermanentFilter
	kind             EventType
	opponent         *PermanentBinding
	savedSubject     *PermanentBinding
	captureOpponent  bool
	duplicateCapture bool
	captureSubject   bool
	duplicateSubject bool
}

// BlocksCreatureMatching checks each opposing creature at the block event.
func BlocksCreatureMatching(subject PermanentRef, filter PermanentFilter) filteredCombatPairPattern {
	return filteredCombatPairPattern{subject: subject, filter: filter, kind: EvtDeclaredBlocker}
}

// BecomesBlockedByCreatureMatching checks each blocker at the block event.
func BecomesBlockedByCreatureMatching(subject PermanentRef, filter PermanentFilter) filteredCombatPairPattern {
	return filteredCombatPairPattern{subject: subject, filter: filter, kind: EvtCreatureBlockedBy}
}

// BlocksOrBecomesBlockedByCreatureMatching checks each new opposing creature.
func BlocksOrBecomesBlockedByCreatureMatching(subject PermanentRef, filter PermanentFilter) filteredCombatPairPattern {
	return filteredCombatPairPattern{subject: subject, filter: filter, kind: EvtCombatPair}
}

// WithOpponent saves the opposing incarnation before trigger placement.
func (p filteredCombatPairPattern) WithOpponent(binding *PermanentBinding) filteredCombatPairPattern {
	if p.captureOpponent {
		p.duplicateCapture = true
	}
	p.opponent, p.captureOpponent = binding, true
	return p
}

// WithSubject saves the watched battlefield visit before trigger placement.
func (p filteredCombatPairPattern) WithSubject(binding *PermanentBinding) filteredCombatPairPattern {
	if p.captureSubject {
		p.duplicateSubject = true
	}
	p.savedSubject, p.captureSubject = binding, true
	return p
}

func (filteredCombatPairPattern) eventPattern() {}

func (p filteredCombatPairPattern) makeTrigger(body ResolvingEffect) *GenericTriggered {
	return NewTriggered(p.kind, false, body).SetConditionData(p)
}

func (p filteredCombatPairPattern) CheckTriggerCond(event *GameEvent, reader GameReader, source, controller uuid.UUID) bool {
	g, ok := reader.(*Game)
	if !ok || p.subject == nil {
		return false
	}
	subject, present := p.subject.resolvePermanent(&EffectContext{Game: g, SourceID: source, Controller: controller})
	return present && p.matchesSubject(event, g, subject)
}

func (p filteredCombatPairPattern) matchesSubject(event *GameEvent, reader *Game, subject *Permanent) bool {
	if subject == nil || subject.PhasedOut || !subject.HasType(TypeCreature) || subject.ID() != event.SourceID {
		return false
	}
	if event.SourceIncarnation != uuid.Nil && event.SourceIncarnation != subject.incarnationID {
		return false
	}
	opponent := reader.FindPermanent(event.TargetID)
	return opponent != nil && !opponent.PhasedOut && opponent.HasType(TypeCreature) &&
		(event.TargetIncarnation == uuid.Nil || event.TargetIncarnation == opponent.incarnationID) && p.filter.Match(opponent, reader)
}

func (p filteredCombatPairPattern) captureDetectedEvent(event *GameEvent, reader GameReader, _ uuid.UUID) []Effect {
	var captures []Effect
	if p.captureSubject && p.savedSubject != nil {
		if subject := reader.FindPermanent(event.SourceID); subject != nil && !subject.PhasedOut {
			captures = append(captures, &eventPermanentCapture{binding: p.savedSubject, value: capturedPermanent{subject.ID(), subject.incarnationID}})
		}
	}
	if p.captureOpponent && p.opponent != nil {
		if opponent := reader.FindPermanent(event.TargetID); opponent != nil && !opponent.PhasedOut {
			captures = append(captures, &eventPermanentCapture{binding: p.opponent, value: capturedPermanent{opponent.ID(), opponent.incarnationID}})
		}
	}
	return captures
}

func (p filteredCombatPairPattern) inspectDefinition() definitionNode {
	n := definitionNode{Role: "event-pattern", Text: "creature combat pair matching " + p.filter.Text(), battlefieldSource: true}
	switch p.subject.(type) {
	case sourceRef, attachedPermanentRef:
	default:
		n.problem = "combat pair subject requires This or AttachedPermanent"
	}
	if p.captureOpponent {
		n.bindingWrite = p.opponent
		n.children = []definitionEdge{{"opponent", &eventPermanentCapture{binding: p.opponent}}}
	}
	if p.captureOpponent && p.opponent == nil {
		n.problem = "combat pair event: supply an opponent binding"
	}
	if p.duplicateCapture {
		n.problem = "combat pair event: save one opponent"
	}
	if p.captureSubject {
		n.bindingWrites = append(n.bindingWrites, p.savedSubject)
		n.children = append(n.children, definitionEdge{"subject", &eventPermanentCapture{binding: p.savedSubject}})
	}
	if p.captureSubject && p.savedSubject == nil {
		n.problem = "combat pair event: supply a subject binding"
	}
	if p.duplicateSubject {
		n.problem = "combat pair event: save one subject"
	}
	if p.captureSubject && p.captureOpponent && p.savedSubject != nil && p.savedSubject == p.opponent {
		n.problem = "combat pair event: use separate subject and opponent bindings"
	}
	return n
}
