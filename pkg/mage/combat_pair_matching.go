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
	captureOpponent  bool
	duplicateCapture bool
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
	if !present || subject.PhasedOut || !subject.HasType(TypeCreature) || subject.ID() != event.SourceID {
		return false
	}
	if event.SourceIncarnation != uuid.Nil && event.SourceIncarnation != subject.incarnationID {
		return false
	}
	opponent := reader.FindPermanent(event.TargetID)
	return opponent != nil && !opponent.PhasedOut && opponent.HasType(TypeCreature) &&
		(event.TargetIncarnation == uuid.Nil || event.TargetIncarnation == opponent.incarnationID) && p.filter.Match(opponent, g)
}

func (p filteredCombatPairPattern) captureDetectedEvent(event *GameEvent, reader GameReader, _ uuid.UUID) []Effect {
	if !p.captureOpponent || p.opponent == nil {
		return nil
	}
	opponent := reader.FindPermanent(event.TargetID)
	if opponent == nil || opponent.PhasedOut {
		return nil
	}
	return []Effect{&eventPermanentCapture{binding: p.opponent, value: capturedPermanent{opponent.ID(), opponent.incarnationID}}}
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
	return n
}
