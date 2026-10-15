package mage

import (
	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

// UntilPermanentHasCounter saves one visit and ends when its counter condition ends.
// A later counter or phase-in cannot start the saved duration again.
func UntilPermanentHasCounter(subject PermanentRef, counter core.CounterType, modifications ...ContinuousModification) ResolvingEffect {
	return &untilEffect{duration: core.Indefinite, counterBound: true, counterSubject: subject, counterType: counter, modifications: append([]ContinuousModification(nil), modifications...)}
}

type capturedCounterLifetime struct {
	subject    capturedPermanent
	counter    core.CounterType
	zeroEpoch  uint64
	phaseEpoch uint64
}

func (c capturedCounterLifetime) active(g *Game) bool {
	p := g.FindPermanent(c.subject.id)
	return p != nil && !p.PhasedOut && p.incarnationID == c.subject.incarnation && p.Counters[c.counter] > 0 && p.counterZeroEpoch[c.counter] == c.zeroEpoch && p.phaseEpoch == c.phaseEpoch
}

func (em *EffectManager) expireCounterDurations(g *Game) bool {
	changed := false
	for _, effect := range em.effects {
		e, ok := effect.(*composedLayerEffect)
		if ok && !e.expired && e.counterLifetime.subject.id != uuid.Nil && !e.counterLifetime.active(g) {
			e.expired = true
			changed = true
		}
	}
	return changed
}
